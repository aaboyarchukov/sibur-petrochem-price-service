-- +goose Up
-- Рыночные котировки по датам. В выгрузке встречаются дубли по (quote_code, quote_date, quote_type),
-- разрешаются по максимальному tech_load_ts — поэтому суррогатный ключ.
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

-- Основной паттерн выборки: по коду котировки и дате периода с каскадом типа Факт -> ОФ -> ППР.
CREATE INDEX quotes_code_date_type_idx ON quotes (quote_code, quote_date, quote_type);

COMMENT ON TABLE quotes IS 'Значения рыночных котировок (источник: quotes.csv)';
COMMENT ON COLUMN quotes.quote_type IS 'Тип публикации: Факт | ОФ | ППР (каскад приоритета при подборе)';
COMMENT ON COLUMN quotes.quote_code IS 'ID котировки в озере данных (связь с quote_mapping.lake_id)';
COMMENT ON COLUMN quotes.tech_load_ts IS 'Метка загрузки; используется для разрешения дублей (берётся самая свежая)';

-- +goose Down
DROP TABLE quotes;
