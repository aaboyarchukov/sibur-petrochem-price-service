-- +goose Up
-- Маппинг SAP-имени котировки на ID в озере данных (quotes.quote_code).
-- Имя котировки может повторяться с разными ID озера — поэтому суррогатный ключ.
CREATE TABLE quote_mapping (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    quote_origin   text,
    quote_kind     text,
    quote_name     text NOT NULL,
    lake_id        bigint,
    lake_id2       text,
    quote_currency text NOT NULL
);

-- Поиск по SAP-имени котировки из компонентов формулы.
CREATE INDEX quote_mapping_quote_name_idx ON quote_mapping (quote_name);

COMMENT ON TABLE quote_mapping IS 'Маппинг котировок: formula_components.quote_name -> quote_mapping.quote_name -> lake_id = quotes.quote_code';
COMMENT ON COLUMN quote_mapping.quote_origin IS 'Происхождение котировки (SAP)';
COMMENT ON COLUMN quote_mapping.quote_name IS 'Имя котировки в SAP (совпадает с formula_components.quote_name)';
COMMENT ON COLUMN quote_mapping.lake_id IS 'ID в озере данных — ключ к quotes.quote_code; может отсутствовать';
COMMENT ON COLUMN quote_mapping.lake_id2 IS 'Альтернативный ID в озере; бывает нечисловым (например AAHQU00)';

-- +goose Down
DROP TABLE quote_mapping;
