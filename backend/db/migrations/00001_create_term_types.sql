-- +goose Up
-- Справочник типов термов формулы (расшифровка кодов SAP из formula_components).
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

-- +goose Down
DROP TABLE term_types;
