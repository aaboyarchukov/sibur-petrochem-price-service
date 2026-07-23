-- +goose Up
-- Курсы валют к рублю по дням. Ключ натуральный: (валюта, день, тип версии) — дублей в данных нет.
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

-- +goose Down
DROP TABLE currency_rates;
