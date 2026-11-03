-- +goose Up

-- Invoices are issued in EUR, the store's only currency since Bulgaria adopted the euro on 1 January 2026.
-- Every new invoice already takes its currency from its order, so this changes only the default for a row
-- written without one. Invoices issued before this migration keep the currency they were issued in.
ALTER TABLE invoices ALTER COLUMN currency SET DEFAULT 'EUR';

-- +goose Down

ALTER TABLE invoices ALTER COLUMN currency SET DEFAULT 'BGN';
