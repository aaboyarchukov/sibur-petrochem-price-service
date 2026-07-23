-- +goose Up
-- Компоненты (термы) договорных формул: переменные выражения и правила получения их значений.
-- В выгрузке есть дубли даже по (formula_id, term_no) — поэтому суррогатный ключ;
-- при расчёте для дублей переменной берётся компонент с самым поздним valid_from.
CREATE TABLE formula_components (
    id             bigint    GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    formula_id     text      NOT NULL,
    term_no        integer   NOT NULL,
    condition_kind text,
    var_name       text      NOT NULL,
    valid_from     timestamp NOT NULL,
    valid_to       timestamp NOT NULL,
    type_code      text      NOT NULL REFERENCES term_types (type_code),
    value          numeric,
    currency       text,
    quote_name     text,
    quote_kind     text,
    calc_rule      text,
    fx_period_rule text,
    period_rule    text,
    term_calc_kind text,
    factor1        numeric,
    factor2        numeric,
    quote_origin   text
);

-- Все компоненты формулы выбираются одним запросом при расчёте строки.
CREATE INDEX formula_components_formula_id_idx ON formula_components (formula_id);

COMMENT ON TABLE formula_components IS 'Состав формул (источник: formula_components.csv). FK на formulas не задан — каталог формул загружается пользователем отдельно';
COMMENT ON COLUMN formula_components.formula_id IS 'Ключ формулы (formulas.Ключ формулы), например Z900026393';
COMMENT ON COLUMN formula_components.term_no IS 'Номер терма в формуле (в выгрузке неуникален в пределах формулы)';
COMMENT ON COLUMN formula_components.var_name IS 'Имя переменной в тексте формулы';
COMMENT ON COLUMN formula_components.type_code IS 'Тип терма (term_types): 1 — котировка, 5 — курс, H/A/B/C/D/E — константы';
COMMENT ON COLUMN formula_components.value IS 'Значение константного терма; NULL трактуется как 0 с предупреждением';
COMMENT ON COLUMN formula_components.quote_name IS 'Для типа 1 — SAP-имя котировки (через quote_mapping); для типа 5 — технический ID валютного терма (ZF...)';
COMMENT ON COLUMN formula_components.fx_period_rule IS 'Правило определения периода для валютного курса (SAP)';
COMMENT ON COLUMN formula_components.period_rule IS 'Правило определения периода значения (SAP)';

-- +goose Down
DROP TABLE formula_components;
