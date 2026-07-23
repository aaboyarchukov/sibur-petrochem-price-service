-- Чтение источников целиком: движок расчёта загружает данные в память.

-- name: ListSsp :many
SELECT row_id, period, customer_asv, customer, mtr_nsi_code, mtr_nsi_name,
       contract, currency, forecast, country, region, market, client_id, client_name
FROM ssp
ORDER BY row_id, period;

-- name: ListFormulas :many
SELECT formula_id, formula_text, material_group_m, business_partner,
       valid_from, valid_to, created_at, inactive, price_type, doc_currency,
       material_id, client_id, client_name
FROM formulas
ORDER BY formula_id;

-- name: ListFormulaComponents :many
SELECT formula_id, term_no, var_name, valid_from, valid_to, type_code,
       value, currency, quote_name
FROM formula_components
ORDER BY formula_id, term_no;

-- name: ListTermTypes :many
SELECT type_code, type_label, category, fixed_term_kind, description, value_source
FROM term_types
ORDER BY type_code;

-- name: ListQuotes :many
SELECT quote_type, quote_name, tech_quote_name, quote_code, quote_date,
       quote_currency, quote_val, tech_load_ts
FROM quotes
ORDER BY quote_code, quote_date;

-- name: ListQuoteMapping :many
SELECT quote_name, lake_id, quote_currency
FROM quote_mapping
ORDER BY quote_name;

-- name: ListCurrencyRates :many
SELECT currency_name, calday, version_type, currency_value
FROM currency_rates
ORDER BY currency_name, calday;

-- name: ListMaterialGroups :many
SELECT group_m, material_id, material_name, valid_from, valid_to
FROM material_groups
ORDER BY group_m, material_id;

-- name: CountSourceRows :one
SELECT
    (SELECT count(*) FROM ssp)                AS ssp,
    (SELECT count(*) FROM formulas)           AS formulas,
    (SELECT count(*) FROM formula_components) AS formula_components,
    (SELECT count(*) FROM term_types)         AS term_types,
    (SELECT count(*) FROM quotes)             AS quotes,
    (SELECT count(*) FROM quote_mapping)      AS quote_mapping,
    (SELECT count(*) FROM currency_rates)     AS currency_rates,
    (SELECT count(*) FROM material_groups)    AS material_groups;
