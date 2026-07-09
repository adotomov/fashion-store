-- +goose Up
-- The NKS (Номенклатурен код на стоката) commodity code turned out not to
-- apply to products — tax group + category internal identifier cover the
-- requirement instead — so it's removed everywhere.
ALTER TABLE invoice_line_items DROP COLUMN nks_code;
ALTER TABLE products DROP COLUMN nks_code;

-- +goose Down
ALTER TABLE products ADD COLUMN nks_code TEXT;
ALTER TABLE invoice_line_items ADD COLUMN nks_code TEXT NOT NULL DEFAULT '';
