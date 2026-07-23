-- Схема справочных таблиц сервиса расчёта цен: типы термов, группы материалов,
-- маппинг котировок, котировки, курсы валют, компоненты формул.

CREATE TABLE term_types (
    type_code       text PRIMARY KEY,
    type_label      text NOT NULL,
    category        text NOT NULL,
    fixed_term_kind text,
    description     text,
    value_source    text
);

COMMENT ON TABLE term_types IS 'Типы термов формулы: котировка (1), курс (5), константы (H/A/B/C/D/E), прайс-лист (7), группировки (0/6)';
COMMENT ON COLUMN term_types.type_code IS 'Код типа терма SAP';
COMMENT ON COLUMN term_types.category IS 'Категория: котировка | курс | константа | прайс-лист | группировка';
COMMENT ON COLUMN term_types.value_source IS 'Откуда берётся значение компонента при расчёте';

CREATE TABLE material_groups (
    group_m       text    NOT NULL,
    material_id   bigint  NOT NULL,
    material_name text    NOT NULL,
    name_lvl_2    text,
    name_lvl_3    text,
    tlevel        text,
    valid_from    date    NOT NULL DEFAULT '1900-01-01',
    valid_to      date    NOT NULL DEFAULT '9999-12-31',
    is_leaf       boolean NOT NULL DEFAULT true,

    PRIMARY KEY (group_m, material_id)
);

CREATE INDEX material_groups_material_id_idx ON material_groups (material_id);

COMMENT ON TABLE material_groups IS 'Справочник состава групп материалов M (источник: material_groups.csv, колонки hname/code_nsi)';
COMMENT ON COLUMN material_groups.group_m IS 'Код группы M (hname), например MT00000116';
COMMENT ON COLUMN material_groups.material_id IS 'Код материала НСИ (code_nsi) — соответствует ssp.mtr_nsi_code';
COMMENT ON COLUMN material_groups.valid_from IS 'Действует с (datuv)';
COMMENT ON COLUMN material_groups.valid_to IS 'Действует по (datub)';

CREATE TABLE quote_mapping (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    quote_origin   text,
    quote_kind     text,
    quote_name     text NOT NULL,
    lake_id        bigint,
    lake_id2       text,
    quote_currency text NOT NULL
);

CREATE INDEX quote_mapping_quote_name_idx ON quote_mapping (quote_name);

COMMENT ON TABLE quote_mapping IS 'Маппинг котировок: formula_components.quote_name -> quote_mapping.quote_name -> lake_id = quotes.quote_code';
COMMENT ON COLUMN quote_mapping.quote_name IS 'Имя котировки в SAP (совпадает с formula_components.quote_name)';
COMMENT ON COLUMN quote_mapping.lake_id IS 'ID в озере данных — ключ к quotes.quote_code; может отсутствовать';
COMMENT ON COLUMN quote_mapping.lake_id2 IS 'Альтернативный ID в озере; бывает нечисловым (например AAHQU00)';

CREATE TABLE quotes (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    quote_type      text      NOT NULL,
    quote_name      text      NOT NULL,
    tech_quote_name text      NOT NULL,
    quote_code      bigint    NOT NULL,
    quote_date      date      NOT NULL,
    quote_currency  text      NOT NULL,
    quote_val       numeric   NOT NULL,
    tech_load_ts    timestamp
);

CREATE INDEX quotes_code_date_type_idx ON quotes (quote_code, quote_date, quote_type);

COMMENT ON TABLE quotes IS 'Значения рыночных котировок (источник: quotes.csv)';
COMMENT ON COLUMN quotes.quote_type IS 'Тип публикации: Факт | ОФ | ППР (каскад приоритета при подборе)';
COMMENT ON COLUMN quotes.quote_code IS 'ID котировки в озере данных (связь с quote_mapping.lake_id)';
COMMENT ON COLUMN quotes.tech_load_ts IS 'Метка загрузки; используется для разрешения дублей (берётся самая свежая)';

CREATE TABLE currency_rates (
    currency_name  text    NOT NULL,
    calday         date    NOT NULL,
    version_type   text    NOT NULL,
    calmonth       date    NOT NULL,
    currency_value numeric NOT NULL,

    PRIMARY KEY (currency_name, calday, version_type)
);

COMMENT ON TABLE currency_rates IS 'Курсы валют к RUB по дням (источник: currency_rates.csv)';
COMMENT ON COLUMN currency_rates.version_type IS 'Тип версии: Факт | ОФ | ППР (каскад приоритета при подборе)';
COMMENT ON COLUMN currency_rates.calday IS 'День курса; при отсутствии на дату берётся ближайший день';
COMMENT ON COLUMN currency_rates.currency_value IS 'Рублей за единицу валюты';

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

CREATE INDEX formula_components_formula_id_idx ON formula_components (formula_id);

COMMENT ON TABLE formula_components IS 'Состав формул (источник: formula_components.csv). FK на formulas не задан — каталог формул загружается пользователем отдельно';
COMMENT ON COLUMN formula_components.formula_id IS 'Ключ формулы (formulas.Ключ формулы), например Z900026393';
COMMENT ON COLUMN formula_components.term_no IS 'Номер терма в формуле (в выгрузке неуникален в пределах формулы)';
COMMENT ON COLUMN formula_components.var_name IS 'Имя переменной в тексте формулы';
COMMENT ON COLUMN formula_components.type_code IS 'Тип терма (term_types): 1 — котировка, 5 — курс, H/A/B/C/D/E — константы';
COMMENT ON COLUMN formula_components.value IS 'Значение константного терма; NULL трактуется как 0 с предупреждением';
COMMENT ON COLUMN formula_components.quote_name IS 'Для типа 1 — SAP-имя котировки (через quote_mapping); для типа 5 — технический ID валютного терма (ZF...)';

