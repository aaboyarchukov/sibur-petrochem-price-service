from __future__ import annotations

import argparse
import ast
import json
import math
import re
from decimal import Decimal, ROUND_HALF_UP
from pathlib import Path
from typing import Any

import pandas as pd


FILES = {
    "ssp": "ssp.csv",
    "formulas": "formulas.csv",
    "components": "formula_components.csv",
    "term_types": "term_types.csv",
    "quote_mapping": "quote_mapping.csv",
    "quotes": "quotes.csv",
    "currency_rates": "currency_rates.csv",
    "material_groups": "material_groups.csv",
}

VERSION_PRIORITY = {"Факт": 0, "ОФ": 1, "ППР": 2}
CONSTANT_TYPES = {"H", "A", "B", "C", "D", "E"}

# Технические ID валютных термов из formula_components.csv.
CURRENCY_TERM_MAP: dict[str, tuple[str, ...]] = {
    "ZF0000000000000002": ("single", "USD"),
    "ZF0000000000000003": ("single", "EUR"),
    "ZF0000000000000024": ("single", "CNY"),
    "ZF0000000000000012": ("cross", "USD", "CNY"),  # CNY за 1 USD
}


class PipelineError(ValueError):
    pass


def read_sources(
    data_dir: Path,
    ssp_file: str = "ssp.csv",
    formulas_file: str = "formulas.csv",
) -> dict[str, pd.DataFrame]:
    files = dict(FILES)
    files["ssp"] = ssp_file
    files["formulas"] = formulas_file
    return {
        name: pd.read_csv(data_dir / file_name)
        for name, file_name in files.items()
    }


def normalize_sources(src: dict[str, pd.DataFrame]) -> dict[str, pd.DataFrame]:
    ssp = src["ssp"].copy()
    ssp["period"] = pd.to_datetime(ssp["period"], errors="raise").dt.normalize()
    ssp["client_id_key"] = ssp["client_id"].astype("string").str.strip()
    ssp["material_id_key"] = pd.to_numeric(ssp["mtr_nsi_code"], errors="raise").astype("Int64")
    ssp["contract_key"] = ssp["contract"].astype("string").str.strip().str.upper()
    # В фактическом файле row_id повторяется по месяцам.
    ssp["demand_key"] = ssp["row_id"].astype(str) + "|" + ssp["period"].dt.strftime("%Y-%m-%d")

    formulas = src["formulas"].copy()
    formulas["formula_id"] = formulas["Ключ формулы"].astype("string").str.strip()
    formulas["client_id_key"] = formulas["client_id"].astype("string").str.strip()
    formulas["formula_material_id"] = pd.to_numeric(formulas["Материал"], errors="coerce").astype("Int64")
    formulas["material_group_m"] = formulas["Подгруппа материалов"].astype("string").str.strip()
    formulas["valid_from"] = pd.to_datetime(
        formulas['"Действительно с" - метка врем. объекта'], errors="raise"
    ).dt.normalize()
    formulas["valid_to"] = pd.to_datetime(
        formulas['Метка времени "Действительно по"'], errors="raise"
    ).dt.normalize()
    formulas["created_at"] = pd.to_datetime(formulas["Дата создания"], errors="raise")
    formulas["inactive"] = formulas["Неактивна"].fillna("").astype(str).str.strip().eq("X")

    groups = src["material_groups"].copy()
    groups["group_m"] = groups["hname"].astype("string").str.strip()
    groups["group_material_id"] = pd.to_numeric(groups["code_nsi"], errors="raise").astype("Int64")
    groups["group_valid_from"] = pd.to_datetime(groups["datuv"], errors="coerce").fillna(pd.Timestamp.min).dt.normalize()
    groups["group_valid_to"] = pd.to_datetime(groups["datub"], errors="coerce").fillna(pd.Timestamp.max.normalize())

    components = src["components"].copy()
    components["formula_id"] = components["Формула"].astype("string").str.strip()
    components["var_name"] = components["Имя переменной"].astype("string").str.strip()
    components["component_type_code"] = components["Тип фиксированного терма"].astype("string").str.strip()
    components["component_valid_from"] = pd.to_datetime(components["Действительно с"], errors="raise").dt.normalize()
    components["component_valid_to"] = pd.to_datetime(components["Действительно по"], errors="raise").dt.normalize()

    term_types = src["term_types"].copy()
    term_types["component_type_code"] = term_types["type_code"].astype("string").str.strip()
    components = components.merge(
        term_types[["component_type_code", "type_label", "category", "value_source"]],
        on="component_type_code",
        how="left",
        validate="many_to_one",
    )

    quote_mapping = src["quote_mapping"].copy()
    quote_mapping["quote_name_key"] = quote_mapping["Имя котировки"].astype("string").str.strip()
    quote_mapping["quote_code_key"] = pd.to_numeric(
        quote_mapping["ID в озере данных"], errors="coerce"
    ).astype("Int64")

    quotes = src["quotes"].copy()
    quotes["quote_code_key"] = pd.to_numeric(quotes["quote_code"], errors="raise").astype("Int64")
    quotes["quote_date"] = pd.to_datetime(quotes["quote_date"], errors="raise").dt.normalize()
    quotes["version_rank"] = quotes["quote_type"].map(VERSION_PRIORITY).fillna(99).astype(int)
    quotes["tech_load_ts"] = pd.to_datetime(quotes["tech_load_ts"], errors="coerce")

    rates = src["currency_rates"].copy()
    rates["currency_name"] = rates["currency_name"].astype("string").str.strip().str.upper()
    rates["calday"] = pd.to_datetime(rates["calday"], errors="raise").dt.normalize()
    rates["version_rank"] = rates["version_type"].map(VERSION_PRIORITY).fillna(99).astype(int)

    return {
        "ssp": ssp,
        "formulas": formulas,
        "material_groups": groups,
        "components": components,
        "quote_mapping": quote_mapping,
        "quotes": quotes,
        "currency_rates": rates,
    }


def build_formula_candidates(
    ssp: pd.DataFrame,
    formulas: pd.DataFrame,
    material_groups: pd.DataFrame,
) -> pd.DataFrame:
    # contract=Formula участвует в подборе. formulas["Тип цены"] — SAP-классификатор и не джойнится с contract.
    demand = ssp.loc[ssp["contract_key"].eq("FORMULA")].copy()
    active = formulas.loc[~formulas["inactive"]].copy()

    # Формулы, заданные прямо на material_id.
    # Берём как актуальные, так и уже завершившиеся формулы. Формулы,
    # которые начнут действовать только после периода спроса, не рассматриваются.
    direct = demand.merge(
        active.loc[active["formula_material_id"].notna()],
        left_on=["client_id_key", "material_id_key"],
        right_on=["client_id_key", "formula_material_id"],
        how="inner",
        suffixes=("_ssp", "_formula"),
    )
    direct = direct.loc[direct["period"].ge(direct["valid_from"])].copy()
    direct["match_scope"] = "material"

    # Формулы на группу M: Подгруппа материалов -> hname -> code_nsi.
    group_formulas = active.loc[active["material_group_m"].notna()].merge(
        material_groups[
            ["group_m", "group_material_id", "group_valid_from", "group_valid_to"]
        ].drop_duplicates(),
        left_on="material_group_m",
        right_on="group_m",
        how="inner",
    )
    grouped = demand.merge(
        group_formulas,
        left_on=["client_id_key", "material_id_key"],
        right_on=["client_id_key", "group_material_id"],
        how="inner",
        suffixes=("_ssp", "_formula"),
    )
    grouped = grouped.loc[
        grouped["period"].ge(grouped["valid_from"])
        & grouped["period"].between(grouped["group_valid_from"], grouped["group_valid_to"])
    ].copy()
    grouped["match_scope"] = "group_m"

    candidates = pd.concat([direct, grouped], ignore_index=True, sort=False)
    if candidates.empty:
        return candidates

    candidates = candidates.drop_duplicates(["demand_key", "formula_id"])
    candidates["is_formula_actual"] = candidates["period"].between(
        candidates["valid_from"], candidates["valid_to"]
    )
    candidates["scope_priority"] = candidates["match_scope"].map(
        {"material": 0, "group_m": 1}
    ).fillna(99).astype(int)

    # candidate_rank остаётся диагностическим порядком до расчёта.
    # Финальный выбор для results выполняется после расчёта всех кандидатов.
    candidates = candidates.sort_values(
        [
            "demand_key",
            "is_formula_actual",
            "scope_priority",
            "created_at",
            "valid_from",
            "valid_to",
            "formula_id",
        ],
        ascending=[True, False, True, False, False, False, True],
    )
    candidates["candidate_rank"] = candidates.groupby("demand_key").cumcount() + 1
    candidates["is_default"] = candidates["candidate_rank"].eq(1)
    candidates["candidate_count"] = candidates.groupby("demand_key")["formula_id"].transform("size")
    return candidates



def build_indexes(src: dict[str, pd.DataFrame]) -> dict[str, Any]:
    quote_name_to_codes = (
        src["quote_mapping"]
        .dropna(subset=["quote_name_key", "quote_code_key"])
        .groupby("quote_name_key")["quote_code_key"]
        .apply(lambda s: list(dict.fromkeys(int(x) for x in s)))
        .to_dict()
    )
    quotes_by_code = {
        int(code): part.copy()
        for code, part in src["quotes"].groupby("quote_code_key", sort=False)
    }
    rates_by_currency = {
        str(currency): part.copy()
        for currency, part in src["currency_rates"].groupby("currency_name", sort=False)
    }
    components_by_formula = {
        str(formula_id): part.copy()
        for formula_id, part in src["components"].groupby("formula_id", sort=False)
    }
    return {
        "quote_name_to_codes": quote_name_to_codes,
        "quotes_by_code": quotes_by_code,
        "rates_by_currency": rates_by_currency,
        "components_by_formula": components_by_formula,
    }


def choose_nearest_version(
    frame: pd.DataFrame,
    target_date: pd.Timestamp,
    date_col: str,
    version_col: str,
    load_ts_col: str | None = None,
) -> pd.Series:
    if frame.empty:
        raise PipelineError("Справочник не содержит подходящих строк")

    work = frame.copy()
    work["_date_gap_days"] = (work[date_col] - target_date).abs().dt.days
    work["_future"] = work[date_col].gt(target_date).astype(int)
    sort_cols = ["_date_gap_days", "version_rank", "_future"]
    ascending = [True, True, True]
    if load_ts_col:
        sort_cols.append(load_ts_col)
        ascending.append(False)
    return work.sort_values(sort_cols, ascending=ascending).iloc[0]


def resolve_quote(
    component: pd.Series,
    period: pd.Timestamp,
    indexes: dict[str, Any],
) -> tuple[float, dict[str, Any]]:
    quote_name = str(component["Имя котировки"]).strip()
    codes = indexes["quote_name_to_codes"].get(quote_name, [])
    if not codes:
        raise PipelineError(f"Нет маппинга котировки: {quote_name}")

    parts = [indexes["quotes_by_code"][code] for code in codes if code in indexes["quotes_by_code"]]
    if not parts:
        raise PipelineError(f"Нет значений котировки: {quote_name}; quote_code={codes}")

    chosen = choose_nearest_version(
        pd.concat(parts, ignore_index=True),
        period,
        date_col="quote_date",
        version_col="quote_type",
        load_ts_col="tech_load_ts",
    )
    return float(chosen["quote_val"]), {
        "source": "quotes.csv",
        "quote_name": quote_name,
        "quote_code": int(chosen["quote_code_key"]),
        "source_date": chosen["quote_date"].date().isoformat(),
        "version_type": chosen["quote_type"],
        "source_currency": chosen["quote_currency"],
        "date_gap_days": int(abs((chosen["quote_date"] - period).days)),
    }


def resolve_rate(
    currency: str,
    period: pd.Timestamp,
    indexes: dict[str, Any],
) -> tuple[float, dict[str, Any]]:
    currency = currency.upper()
    if currency == "RUB":
        return 1.0, {
            "source": "currency_rates.csv",
            "currency": "RUB",
            "source_date": period.date().isoformat(),
            "version_type": "identity",
            "date_gap_days": 0,
        }

    frame = indexes["rates_by_currency"].get(currency)
    if frame is None:
        raise PipelineError(f"Нет курса валюты: {currency}")

    chosen = choose_nearest_version(
        frame,
        period,
        date_col="calday",
        version_col="version_type",
    )
    return float(chosen["currency_value"]), {
        "source": "currency_rates.csv",
        "currency": currency,
        "source_date": chosen["calday"].date().isoformat(),
        "version_type": chosen["version_type"],
        "date_gap_days": int(abs((chosen["calday"] - period).days)),
    }


def resolve_currency_component(
    component: pd.Series,
    period: pd.Timestamp,
    indexes: dict[str, Any],
) -> tuple[float, dict[str, Any]]:
    technical_id = "" if pd.isna(component["Имя котировки"]) else str(component["Имя котировки"]).strip()
    variable = str(component["var_name"]).strip().upper()
    explicit_currency = "" if pd.isna(component["Валюта"]) else str(component["Валюта"]).strip().upper()

    if explicit_currency in indexes["rates_by_currency"] or explicit_currency == "RUB":
        return resolve_rate(explicit_currency, period, indexes)

    if technical_id in CURRENCY_TERM_MAP:
        mode, *currencies = CURRENCY_TERM_MAP[technical_id]
        if mode == "single":
            value, meta = resolve_rate(currencies[0], period, indexes)
            meta["technical_id"] = technical_id
            return value, meta
        numerator, meta_num = resolve_rate(currencies[0], period, indexes)
        denominator, meta_den = resolve_rate(currencies[1], period, indexes)
        return numerator / denominator, {
            "source": "currency_rates.csv",
            "technical_id": technical_id,
            "currency_pair": f"{currencies[0]}/{currencies[1]}",
            "numerator": meta_num,
            "denominator": meta_den,
        }

    if variable in {"USD", "EUR", "CNY", "RUB"}:
        return resolve_rate(variable, period, indexes)

    raise PipelineError(
        f"Неизвестный валютный терм: var={component['var_name']}; technical_id={technical_id}"
    )


def resolve_components(
    formula_id: str,
    period: pd.Timestamp,
    indexes: dict[str, Any],
) -> tuple[dict[str, float], list[dict[str, Any]], list[str]]:
    frame = indexes["components_by_formula"].get(formula_id)
    if frame is None:
        return {}, [], [f"Нет компонентов для формулы {formula_id}"]

    # Оставляем компоненты, действующие в период строки; для дублей переменной — самый поздний valid_from.
    active = frame.loc[
        frame["component_valid_from"].le(period) & frame["effective_component_valid_to"].ge(period)
    ].copy()
    active = active.sort_values(
        ["var_name", "component_valid_from", "Номер терма в формуле"],
        ascending=[True, False, False],
    ).drop_duplicates("var_name")

    values: dict[str, float] = {}
    details: list[dict[str, Any]] = []
    errors: list[str] = []

    for _, component in active.iterrows():
        var_name = str(component["var_name"])
        type_code = str(component["component_type_code"])
        detail: dict[str, Any] = {
            "formula_id": formula_id,
            "period": period.date().isoformat(),
            "var_name": var_name,
            "component_type_code": type_code,
            "component_type_label": component.get("type_label"),
        }
        try:
            if type_code in CONSTANT_TYPES:
                if pd.isna(component["Значение"]):
                    # В выгрузке Pricing пустой фиксированный терм означает
                    # отсутствие надбавки/скидки. Для расчёта используем 0,
                    # но сохраняем предупреждение в детализации.
                    value = 0.0
                    meta = {
                        "source": "formula_components.csv",
                        "raw_value": None,
                        "warning": "Пустой фиксированный терм принят равным 0",
                    }
                else:
                    value = float(component["Значение"])
                    meta = {
                        "source": "formula_components.csv",
                        "raw_value": component["Значение"],
                    }
            elif type_code == "1":
                value, meta = resolve_quote(component, period, indexes)
            elif type_code == "5":
                value, meta = resolve_currency_component(component, period, indexes)
            elif type_code == "7":
                raise PipelineError("Тип 7 требует отдельного источника Pricing/SAP, которого нет в наборе")
            else:
                raise PipelineError(f"Неподдерживаемый тип компонента: {type_code}")

            values[var_name] = value
            detail.update({"value": value, "status": "OK", **meta})
        except Exception as exc:
            message = f"{var_name}: {exc}"
            errors.append(message)
            detail.update({"value": None, "status": "ERROR", "error": str(exc)})
        details.append(detail)

    return values, details, errors


def commercial_round(value: float, digits: float = 0) -> float:
    places = int(digits)
    quantum = Decimal("1").scaleb(-places)
    return float(Decimal(str(value)).quantize(quantum, rounding=ROUND_HALF_UP))


def if_func(condition: Any, when_true: Any, when_false: Any = 0) -> Any:
    return when_true if bool(condition) else when_false


ALLOWED_FUNCTIONS = {
    "IF": if_func,
    "RND_X": commercial_round,
    "MIN": min,
    "MAX": max,
}
ALLOWED_AST_NODES = (
    ast.Expression,
    ast.Constant,
    ast.Name,
    ast.Load,
    ast.BinOp,
    ast.UnaryOp,
    ast.Compare,
    ast.Call,
    ast.Add,
    ast.Sub,
    ast.Mult,
    ast.Div,
    ast.Mod,
    ast.Pow,
    ast.UAdd,
    ast.USub,
    ast.Gt,
    ast.GtE,
    ast.Lt,
    ast.LtE,
    ast.Eq,
    ast.NotEq,
)
TOKEN_CHARS = r"A-Za-z0-9_$€¥₱"


def evaluate_formula(expression: str, variables: dict[str, float]) -> float:
    translated = str(expression)
    aliases: dict[str, float] = {}

    # Переменные могут содержать $, €, ¥, ₱ и начинаться с цифры, поэтому заменяем их на безопасные alias.
    for index, var_name in enumerate(sorted(variables, key=len, reverse=True)):
        alias = f"v_{index}"
        pattern = rf"(?<![{TOKEN_CHARS}]){re.escape(var_name)}(?![{TOKEN_CHARS}])"
        translated, count = re.subn(pattern, alias, translated)
        if count:
            aliases[alias] = float(variables[var_name])

    try:
        tree = ast.parse(translated, mode="eval")
    except SyntaxError as exc:
        raise PipelineError(f"Синтаксическая ошибка формулы: {exc.msg}") from exc

    for node in ast.walk(tree):
        if not isinstance(node, ALLOWED_AST_NODES):
            raise PipelineError(f"Запрещённый элемент формулы: {type(node).__name__}")
        if isinstance(node, ast.Name) and node.id not in aliases and node.id not in ALLOWED_FUNCTIONS:
            raise PipelineError(f"Не найдена переменная: {node.id}")
        if isinstance(node, ast.Call):
            if not isinstance(node.func, ast.Name) or node.func.id not in ALLOWED_FUNCTIONS:
                raise PipelineError("Запрещённая функция")

    value = eval(
        compile(tree, "<formula>", "eval"),
        {"__builtins__": {}},
        {**ALLOWED_FUNCTIONS, **aliases},
    )
    if isinstance(value, bool) or not isinstance(value, (int, float)) or not math.isfinite(float(value)):
        raise PipelineError(f"Формула вернула некорректное значение: {value}")
    return float(value)


def convert_currency(
    value: float,
    source_currency: str,
    target_currency: str,
    period: pd.Timestamp,
    indexes: dict[str, Any],
) -> tuple[float, dict[str, Any]]:
    source_currency = str(source_currency).upper()
    target_currency = str(target_currency).upper()
    if source_currency == target_currency:
        return value, {"conversion": "identity"}

    source_rate, source_meta = resolve_rate(source_currency, period, indexes)
    target_rate, target_meta = resolve_rate(target_currency, period, indexes)
    return value * source_rate / target_rate, {
        "conversion": f"{source_currency}->{target_currency}",
        "source_rate": source_meta,
        "target_rate": target_meta,
    }


def calculate_candidates(
    candidates: pd.DataFrame,
    indexes: dict[str, Any],
) -> tuple[pd.DataFrame, pd.DataFrame]:
    candidate_rows: list[dict[str, Any]] = []
    component_rows: list[dict[str, Any]] = []

    for _, row in candidates.iterrows():
        formula_id = str(row["formula_id"])
        period = pd.Timestamp(row["period"])
        values, details, component_errors = resolve_components(formula_id, period, indexes)

        for detail in details:
            detail.update({"demand_key": row["demand_key"], "row_id": row["row_id"]})
            component_rows.append(detail)

        is_formula_actual = bool(row["is_formula_actual"])
        result: dict[str, Any] = {
            "demand_key": row["demand_key"],
            "row_id": row["row_id"],
            "period": period,
            "formula_id": formula_id,
            "formula_expression": row["Формула"],
            "formula_currency": row["ВалютаДокумента"],
            "demand_currency": row["currency"],
            "match_scope": row["match_scope"],
            "scope_priority": row["scope_priority"],
            "created_at": row["created_at"],
            "valid_from": row["valid_from"],
            "valid_to": row["valid_to"],
            "is_formula_actual": is_formula_actual,
            "candidate_rank": row["candidate_rank"],
            "candidate_count": row["candidate_count"],
            "is_default": row["is_default"],
            "component_values_json": json.dumps(values, ensure_ascii=False),
            "warning": (
                "Формула неактуальна на дату расчёта"
                if not is_formula_actual
                else None
            ),
        }

        if component_errors:
            result.update(
                {
                    "status": "COMPONENT_ERROR",
                    "price_formula_currency": None,
                    "price": None,
                    "error": " | ".join(component_errors),
                }
            )
            candidate_rows.append(result)
            continue

        try:
            price_formula_currency = evaluate_formula(row["Формула"], values)
            price, conversion_meta = convert_currency(
                price_formula_currency,
                row["ВалютаДокумента"],
                row["currency"],
                period,
                indexes,
            )
            result.update(
                {
                    "status": "CALCULATED",
                    "price_formula_currency": price_formula_currency,
                    "price": price,
                    "currency_conversion_json": json.dumps(conversion_meta, ensure_ascii=False),
                    "error": None,
                }
            )
        except Exception as exc:
            result.update(
                {
                    "status": "INVALID_FORMULA",
                    "price_formula_currency": None,
                    "price": None,
                    "error": str(exc),
                }
            )
        candidate_rows.append(result)

    return pd.DataFrame(candidate_rows), pd.DataFrame(component_rows)


def _combine_messages(*messages: Any) -> str | None:
    parts = [str(message).strip() for message in messages if pd.notna(message) and str(message).strip()]
    return " | ".join(parts) if parts else None


def select_candidate_results(
    candidate_results: pd.DataFrame,
) -> tuple[pd.DataFrame, pd.DataFrame]:
    """Выбирает итоговую формулу после расчёта всех кандидатов."""
    if candidate_results.empty:
        return candidate_results.copy(), candidate_results.copy()

    work = candidate_results.copy()
    work["is_selected"] = False
    work["selection_reason"] = pd.NA
    work["selection_status"] = pd.NA
    work["equal_priority_count"] = 0
    work["requires_review"] = False

    selected_records: list[dict[str, Any]] = []
    conflict_message = (
        "Найдено несколько равноприоритетных формул. "
        "Автоматически выбрана формула с минимальным formula_id. "
        "Нужно выбрать формулу"
    )

    for _, group in work.groupby("demand_key", sort=False):
        successful = group.loc[group["status"].eq("CALCULATED")].copy()
        actual_successful = successful.loc[successful["is_formula_actual"]].copy()

        if not actual_successful.empty:
            pool = actual_successful.sort_values(
                ["scope_priority", "created_at", "valid_from", "formula_id"],
                ascending=[True, False, False, True],
            )
            top = pool.iloc[0]
            equal_priority = pool.loc[
                pool["scope_priority"].eq(top["scope_priority"])
                & pool["created_at"].eq(top["created_at"])
                & pool["valid_from"].eq(top["valid_from"])
            ]
            final_status = "CALCULATED"
            selection_reason = "ACTUAL_SUCCESSFUL_FORMULA"
            warning = top.get("warning")
        else:
            expired_successful = successful.loc[~successful["is_formula_actual"]].copy()
            if not expired_successful.empty:
                pool = expired_successful.sort_values(
                    ["valid_to", "scope_priority", "created_at", "valid_from", "formula_id"],
                    ascending=[False, True, False, False, True],
                )
                top = pool.iloc[0]
                equal_priority = pool.loc[
                    pool["valid_to"].eq(top["valid_to"])
                    & pool["scope_priority"].eq(top["scope_priority"])
                    & pool["created_at"].eq(top["created_at"])
                    & pool["valid_from"].eq(top["valid_from"])
                ]
                final_status = "CALCULATED_WITH_EXPIRED_FORMULA"
                selection_reason = "LATEST_EXPIRED_SUCCESSFUL_FORMULA"
                warning = (
                    "Использована неактуальная формула: "
                    f"срок действия закончился {pd.Timestamp(top['valid_to']).strftime('%d.%m.%Y')}, "
                    f"дата расчёта — {pd.Timestamp(top['period']).strftime('%d.%m.%Y')}."
                )
            else:
                # Если ни одна формула не рассчиталась, показываем ошибку
                # наиболее приоритетного кандидата и оставляем цену пустой.
                actual_errors = group.loc[group["is_formula_actual"]].copy()
                if not actual_errors.empty:
                    pool = actual_errors.sort_values(
                        ["scope_priority", "created_at", "valid_from", "formula_id"],
                        ascending=[True, False, False, True],
                    )
                else:
                    pool = group.sort_values(
                        ["valid_to", "scope_priority", "created_at", "valid_from", "formula_id"],
                        ascending=[False, True, False, False, True],
                    )
                top = pool.iloc[0]
                equal_priority = pool.iloc[[0]]
                final_status = str(top["status"])
                selection_reason = "NO_SUCCESSFUL_FORMULA"
                warning = top.get("warning")

        selected_index = top.name
        equal_priority_count = len(equal_priority)
        requires_review = equal_priority_count > 1 and str(top["status"]) == "CALCULATED"
        final_error = top.get("error")

        if requires_review:
            final_status = "CALCULATED_WITH_FORMULA_CONFLICT"
            selection_reason = "TECHNICAL_TIE_BREAK"
            final_error = conflict_message
            warning = _combine_messages(warning, conflict_message)

        record = top.to_dict()
        record.update(
            {
                "status": final_status,
                "error": final_error,
                "warning": warning,
                "selection_reason": selection_reason,
                "equal_priority_count": equal_priority_count,
                "requires_review": requires_review,
            }
        )
        selected_records.append(record)

        work.loc[selected_index, "is_selected"] = True
        work.loc[selected_index, "selection_reason"] = selection_reason
        work.loc[selected_index, "selection_status"] = final_status
        work.loc[selected_index, "equal_priority_count"] = equal_priority_count
        work.loc[selected_index, "requires_review"] = requires_review

    return pd.DataFrame(selected_records), work



def build_results(ssp: pd.DataFrame, selected_results: pd.DataFrame) -> pd.DataFrame:
    base_columns = [
        "demand_key",
        "row_id",
        "period",
        "client_id",
        "client_name",
        "mtr_nsi_code",
        "mtr_nsi_name",
        "contract",
        "currency",
        "forecast",
        "customer_asv",
        "customer",
        "country",
        "region",
        "market",
    ]
    results = ssp[base_columns].copy()

    if selected_results.empty:
        results["candidate_count"] = 0
        results["formula_id"] = pd.NA
        results["formula_expression"] = pd.NA
        results["price"] = pd.NA
        results["status"] = results["contract"].astype(str).str.upper().map(
            lambda x: "SPOT_NOT_CALCULATED" if x == "SPOT" else "FORMULA_NOT_FOUND"
        )
        results["error"] = pd.NA
        return results

    results = results.merge(
        selected_results[
            [
                "demand_key",
                "candidate_count",
                "formula_id",
                "formula_expression",
                "formula_currency",
                "match_scope",
                "created_at",
                "valid_from",
                "valid_to",
                "is_formula_actual",
                "price_formula_currency",
                "price",
                "status",
                "selection_reason",
                "equal_priority_count",
                "requires_review",
                "warning",
                "error",
            ]
        ],
        on="demand_key",
        how="left",
        validate="one_to_one",
    )

    is_spot = results["contract"].astype(str).str.strip().str.upper().eq("SPOT")
    no_formula = results["formula_id"].isna() & ~is_spot
    results.loc[is_spot, "status"] = "SPOT_NOT_CALCULATED"
    results.loc[no_formula, "status"] = "FORMULA_NOT_FOUND"
    results["candidate_count"] = results["candidate_count"].fillna(0).astype(int)
    results["equal_priority_count"] = results["equal_priority_count"].fillna(0).astype(int)
    results["requires_review"] = results["requires_review"].map(lambda value: bool(value) if pd.notna(value) else False)
    return results



def run_pipeline(
    data_dir: Path,
    output_dir: Path,
    ssp_file: str = "ssp.csv",
    formulas_file: str = "formulas.csv",
) -> None:
    src = normalize_sources(read_sources(data_dir, ssp_file=ssp_file, formulas_file=formulas_file))
    horizon_end = src["ssp"]["period"].max()

    # Даты действия формул не изменяются: актуальность определяется строго
    # по valid_from <= period <= valid_to. Компоненты оставлены в прежней
    # логике, чтобы можно было рассчитать разрешённый fallback на завершившуюся формулу.
    src["components"]["effective_component_valid_to"] = src["components"]["component_valid_to"].where(
        src["components"]["component_valid_to"].ge(horizon_end), horizon_end
    )

    candidates = build_formula_candidates(src["ssp"], src["formulas"], src["material_groups"])
    indexes = build_indexes(src)
    candidate_results_raw, component_values = calculate_candidates(candidates, indexes)
    selected_results, candidate_results = select_candidate_results(candidate_results_raw)
    results = build_results(src["ssp"], selected_results)

    output_dir.mkdir(parents=True, exist_ok=True)
    results.to_csv(output_dir / "results.csv", index=False)
    candidate_results.to_csv(output_dir / "formula_candidates.csv", index=False)
    component_values.to_csv(output_dir / "component_values.csv", index=False)

    print(results["status"].value_counts(dropna=False).to_string())
    print(f"Saved: {output_dir}/results.csv, formula_candidates.csv, component_values.csv")



def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--data-dir", type=Path, default=Path("."))
    parser.add_argument("--ssp-file", default="ssp.csv")
    parser.add_argument("--formulas-file", default="formulas.csv")
    parser.add_argument("--output-dir", type=Path, default=Path("price_results"))
    args = parser.parse_args()
    run_pipeline(
        args.data_dir,
        args.output_dir,
        ssp_file=args.ssp_file,
        formulas_file=args.formulas_file,
    )


if __name__ == "__main__":
    main()
