-- +goose Up
CREATE TABLE store_settings (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	store_name TEXT NOT NULL DEFAULT 'My Store',
	locale TEXT NOT NULL DEFAULT 'en-US',
	currency TEXT NOT NULL DEFAULT 'USD',
	address_line1 TEXT,
	address_line2 TEXT,
	city TEXT,
	region TEXT,
	postal_code TEXT,
	country TEXT,
	contact_email TEXT,
	contact_phone TEXT,
	company_description TEXT,
	logo_bucket TEXT,
	logo_object_key TEXT,
	logo_content_type TEXT,
	logo_size_bytes BIGINT,
	terms_bucket TEXT,
	terms_object_key TEXT,
	terms_content_type TEXT,
	terms_size_bytes BIGINT,
	terms_filename TEXT,
	privacy_bucket TEXT,
	privacy_object_key TEXT,
	privacy_content_type TEXT,
	privacy_size_bytes BIGINT,
	privacy_filename TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- store_settings is a singleton: exactly one row, seeded here, never created
-- or deleted by application code (only updated).
INSERT INTO store_settings (store_name, locale, currency) VALUES ('Maison', 'en-US', 'EUR');

-- +goose Down
DROP TABLE store_settings;
