-- +goose Up

-- Configurable invoice settings (singleton row, same pattern as hero_settings)
CREATE TABLE invoice_settings (
    id                 BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
    company_name       TEXT NOT NULL DEFAULT '',
    company_legal_type TEXT NOT NULL DEFAULT 'ООД',
    company_eik        TEXT NOT NULL DEFAULT '',
    company_address    TEXT NOT NULL DEFAULT '',
    company_email      TEXT NOT NULL DEFAULT '',
    company_phone      TEXT NOT NULL DEFAULT '',
    nra_store_number   TEXT NOT NULL DEFAULT '',
    vat_number         TEXT NOT NULL DEFAULT '',
    vat_rate           NUMERIC(5,2) NOT NULL DEFAULT 20.00,
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO invoice_settings DEFAULT VALUES;

-- Configurable courier list (referenced when generating invoices for COD/EasyBox orders)
CREATE TABLE invoice_couriers (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    identifier TEXT NOT NULL UNIQUE,
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO invoice_couriers (name, identifier, sort_order) VALUES
    ('Speedy', 'speedy', 1),
    ('Econt', 'econt', 2),
    ('DHL', 'dhl', 3),
    ('DPD', 'dpd', 4);

-- 10-digit sequential invoice number, separate from internal order IDs
CREATE SEQUENCE invoice_number_seq START 1 INCREMENT 1;

-- Trigger function enforcing immutability on invoices and line items
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION invoices_read_only()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'invoices are immutable - modifications and deletions are not permitted';
END;
$$;
-- +goose StatementEnd

-- Main invoices table — full snapshot of all legally required fields at issuance time
CREATE TABLE invoices (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_number          VARCHAR(10) NOT NULL UNIQUE
                                DEFAULT LPAD(nextval('invoice_number_seq')::text, 10, '0'),
    document_type           TEXT NOT NULL CHECK (document_type IN ('фактура', 'сторно')),
    order_id                UUID NOT NULL REFERENCES orders(id),
    storno_of_invoice_id    UUID REFERENCES invoices(id),

    -- Snapshot of the order at issuance time
    order_number            TEXT NOT NULL,
    placed_at               TIMESTAMPTZ NOT NULL,

    -- Payment reference fields (card: provider + reference; logistics: courier name/id)
    payment_method          TEXT NOT NULL,
    card_provider           TEXT,
    card_provider_reference TEXT,
    courier_name            TEXT,
    courier_identifier      TEXT,

    -- Snapshot of seller (company) details at issuance time
    company_name            TEXT NOT NULL,
    company_legal_type      TEXT NOT NULL,
    company_eik             TEXT NOT NULL,
    company_address         TEXT NOT NULL,
    company_email           TEXT NOT NULL,
    company_phone           TEXT NOT NULL,
    nra_store_number        TEXT NOT NULL,
    vat_number              TEXT NOT NULL,
    vat_rate                NUMERIC(5,2) NOT NULL DEFAULT 20.00,

    -- Buyer (recipient) snapshot
    recipient_name          TEXT NOT NULL,
    recipient_address       TEXT NOT NULL,
    recipient_email         TEXT NOT NULL,

    -- Monetary totals in minor currency units (e.g. stotinki for BGN)
    subtotal_excl_vat_minor BIGINT NOT NULL,
    vat_amount_minor        BIGINT NOT NULL,
    total_incl_vat_minor    BIGINT NOT NULL,
    currency                TEXT NOT NULL DEFAULT 'BGN',
    delivery_fee_minor      BIGINT NOT NULL DEFAULT 0,
    discount_amount_minor   BIGINT,

    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER invoices_no_update
    BEFORE UPDATE ON invoices
    FOR EACH ROW EXECUTE FUNCTION invoices_read_only();

CREATE TRIGGER invoices_no_delete
    BEFORE DELETE ON invoices
    FOR EACH ROW EXECUTE FUNCTION invoices_read_only();

-- Line items: per-item VAT breakdown snapshot
CREATE TABLE invoice_line_items (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id                UUID NOT NULL REFERENCES invoices(id),
    product_name              TEXT NOT NULL,
    variant_label             TEXT NOT NULL DEFAULT '',
    nks_code                  TEXT NOT NULL DEFAULT '',
    quantity                  INT NOT NULL,
    unit_price_incl_vat_minor BIGINT NOT NULL,
    unit_price_excl_vat_minor BIGINT NOT NULL,
    vat_per_unit_minor        BIGINT NOT NULL,
    line_total_incl_vat_minor BIGINT NOT NULL,
    line_total_excl_vat_minor BIGINT NOT NULL,
    line_vat_amount_minor     BIGINT NOT NULL,
    sort_order                INT NOT NULL DEFAULT 0,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER invoice_line_items_no_update
    BEFORE UPDATE ON invoice_line_items
    FOR EACH ROW EXECUTE FUNCTION invoices_read_only();

CREATE TRIGGER invoice_line_items_no_delete
    BEFORE DELETE ON invoice_line_items
    FOR EACH ROW EXECUTE FUNCTION invoices_read_only();

-- Audit log: insert-only record of every significant invoicing event
CREATE TABLE invoice_audit_log (
    id             BIGSERIAL PRIMARY KEY,
    invoice_number TEXT NOT NULL,
    event_type     TEXT NOT NULL,
    actor          TEXT,
    metadata       JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX invoice_audit_log_invoice_number_idx ON invoice_audit_log(invoice_number);

-- +goose Down
DROP TABLE IF EXISTS invoice_audit_log;
DROP TABLE IF EXISTS invoice_line_items;
DROP TABLE IF EXISTS invoices;
DROP SEQUENCE IF EXISTS invoice_number_seq;
DROP FUNCTION IF EXISTS invoices_read_only();
DROP TABLE IF EXISTS invoice_couriers;
DROP TABLE IF EXISTS invoice_settings;
