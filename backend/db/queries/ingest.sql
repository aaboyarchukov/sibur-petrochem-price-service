-- Замещение пользовательских источников при загрузке .xlsx:
-- delete + copyfrom выполняются в одной транзакции (repository).

-- name: DeleteSsp :exec
DELETE FROM ssp;

-- name: DeleteFormulas :exec
DELETE FROM formulas;

-- name: CopySsp :copyfrom
INSERT INTO ssp (
    row_id, period, customer_asv, customer, mtr_nsi_code, mtr_nsi_name,
    contract, currency, forecast, country, region, market, client_id, client_name
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);

-- name: CopyFormulas :copyfrom
INSERT INTO formulas (
    formula_id, formula_text, material_group_m, business_partner,
    valid_from, valid_to, created_at, inactive, price_type, doc_currency,
    material_id, client_id, client_name
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);
