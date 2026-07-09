-- +goose Up

-- VAT tax groups (Bulgarian fiscal groups А–Ж). Each product references one;
-- the group's rate drives per-line VAT on generated invoices.
CREATE TABLE tax_groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identifier  TEXT NOT NULL UNIQUE,
    vat_rate    NUMERIC(5,2) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed the common case: group Б = 20% VAT.
INSERT INTO tax_groups (identifier, vat_rate) VALUES ('Б', 20.00);

ALTER TABLE products ADD COLUMN tax_group_id UUID REFERENCES tax_groups (id);

-- Per-line VAT rate snapshot on invoices (existing rows default to 20%, which
-- matches the historical hardcoded split). The immutability triggers only
-- block UPDATE/DELETE, so ADD COLUMN is permitted.
ALTER TABLE invoice_line_items ADD COLUMN vat_rate NUMERIC(5,2) NOT NULL DEFAULT 20.00;

-- +goose Down
ALTER TABLE invoice_line_items DROP COLUMN vat_rate;
ALTER TABLE products DROP COLUMN tax_group_id;
DROP TABLE tax_groups;
