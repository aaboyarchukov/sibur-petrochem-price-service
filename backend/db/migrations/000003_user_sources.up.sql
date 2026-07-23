-- Пользовательские источники: прогноз спроса (ssp) и каталог формул (formulas).
-- В MVP наполняются seed-миграцией из documents/*.csv; загрузка .xlsx — следующая итерация.

CREATE TABLE ssp (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    row_id       bigint  NOT NULL,
    period       date    NOT NULL,
    customer_asv text,
    customer     text,
    mtr_nsi_code bigint  NOT NULL,
    mtr_nsi_name text    NOT NULL,
    contract     text    NOT NULL,
    currency     text    NOT NULL,
    forecast     numeric,
    country      text,
    region       text,
    market       text,
    client_id    text    NOT NULL,
    client_name  text    NOT NULL,

    -- row_id повторяется по месяцам: уникальна пара (row_id, period)
    UNIQUE (row_id, period)
);

COMMENT ON TABLE ssp IS 'Прогноз спроса (источник: ssp.csv). Строка = клиент+продукт+месяц+объём';
COMMENT ON COLUMN ssp.contract IS 'Тип сделки: Formula (есть контракт) | SPOT (без контракта)';
COMMENT ON COLUMN ssp.mtr_nsi_code IS 'Код материала НСИ — соответствует material_groups.material_id';
COMMENT ON COLUMN ssp.forecast IS 'Объём прогноза, т';

CREATE TABLE formulas (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    formula_id       text      NOT NULL,
    formula_text     text      NOT NULL,
    material_group_m text,
    business_partner text,
    valid_from       date      NOT NULL,
    valid_to         date      NOT NULL,
    created_at       timestamp NOT NULL,
    inactive         boolean   NOT NULL DEFAULT false,
    price_type       text,
    doc_currency     text      NOT NULL,
    material_id      bigint,
    client_id        text      NOT NULL,
    client_name      text      NOT NULL
);

CREATE INDEX formulas_client_material_idx ON formulas (client_id, material_id);
CREATE INDEX formulas_client_group_idx ON formulas (client_id, material_group_m);

COMMENT ON TABLE formulas IS 'Каталог договорных формул (источник: formulas.csv)';
COMMENT ON COLUMN formulas.formula_id IS 'Ключ формулы SAP (Ключ формулы), например Z900026393';
COMMENT ON COLUMN formulas.material_group_m IS 'Подгруппа материалов (группа M); либо группа, либо material_id';
COMMENT ON COLUMN formulas.price_type IS 'Тип цены — SAP-классификатор (A1,1,2…); НЕ джойнится с ssp.contract';
COMMENT ON COLUMN formulas.inactive IS 'Признак «Неактивна» (X в выгрузке)';
COMMENT ON COLUMN formulas.created_at IS 'Дата создания в SAP — критерий выбора формулы по умолчанию';
