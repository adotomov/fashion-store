-- +goose Up
ALTER TABLE invoice_settings
    ADD COLUMN company_address_street      TEXT NOT NULL DEFAULT '',
    ADD COLUMN company_address_city        TEXT NOT NULL DEFAULT '',
    ADD COLUMN company_address_postal_code TEXT NOT NULL DEFAULT '',
    ADD COLUMN company_address_country     TEXT NOT NULL DEFAULT 'България';

-- Preserve any existing value in the street field (best-effort)
UPDATE invoice_settings SET company_address_street = company_address WHERE company_address <> '';

ALTER TABLE invoice_settings DROP COLUMN company_address;

-- +goose Down
ALTER TABLE invoice_settings ADD COLUMN company_address TEXT NOT NULL DEFAULT '';

UPDATE invoice_settings SET company_address = TRIM(CONCAT_WS(', ',
    NULLIF(company_address_street, ''),
    NULLIF(TRIM(company_address_postal_code || ' ' || company_address_city), ''),
    NULLIF(company_address_country, '')
));

ALTER TABLE invoice_settings
    DROP COLUMN company_address_street,
    DROP COLUMN company_address_city,
    DROP COLUMN company_address_postal_code,
    DROP COLUMN company_address_country;
