-- +goose Up

ALTER TABLE store_settings ADD COLUMN legal_entity_name TEXT;

CREATE TABLE store_addresses (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	store_settings_id UUID NOT NULL REFERENCES store_settings(id) ON DELETE CASCADE,
	label TEXT NOT NULL DEFAULT '',
	line1 TEXT NOT NULL DEFAULT '',
	line2 TEXT,
	city TEXT,
	region TEXT,
	postal_code TEXT,
	country TEXT,
	is_default BOOLEAN NOT NULL DEFAULT false,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Carry forward the single legacy address (if any field was set) as the first,
-- default address so existing data isn't lost when address becomes multi-row.
INSERT INTO store_addresses (store_settings_id, label, line1, line2, city, region, postal_code, country, is_default)
SELECT id, 'Main', COALESCE(address_line1, ''), address_line2, city, region, postal_code, country, true
FROM store_settings
WHERE address_line1 IS NOT NULL OR city IS NOT NULL OR country IS NOT NULL;

ALTER TABLE store_settings
	DROP COLUMN address_line1,
	DROP COLUMN address_line2,
	DROP COLUMN city,
	DROP COLUMN region,
	DROP COLUMN postal_code,
	DROP COLUMN country;

CREATE TABLE store_languages (
	code TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	is_default BOOLEAN NOT NULL DEFAULT false,
	enabled BOOLEAN NOT NULL DEFAULT true,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO store_languages (code, name, is_default, enabled) VALUES ('en', 'English', true, true);

CREATE TABLE translations (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	entity_type TEXT NOT NULL,
	entity_id UUID NOT NULL,
	locale TEXT NOT NULL REFERENCES store_languages(code) ON DELETE CASCADE,
	field TEXT NOT NULL,
	value TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE (entity_type, entity_id, locale, field)
);

CREATE INDEX idx_translations_lookup ON translations (entity_type, entity_id, locale);

CREATE TABLE ui_strings (
	key TEXT NOT NULL,
	locale TEXT NOT NULL REFERENCES store_languages(code) ON DELETE CASCADE,
	value TEXT NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (key, locale)
);

CREATE TABLE store_documents (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	type TEXT NOT NULL, -- 'terms' | 'privacy'
	locale TEXT NOT NULL REFERENCES store_languages(code) ON DELETE CASCADE,
	bucket TEXT NOT NULL,
	object_key TEXT NOT NULL,
	content_type TEXT NOT NULL,
	size_bytes BIGINT NOT NULL,
	filename TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE (type, locale)
);

-- Carry forward existing single-locale terms/privacy uploads into store_documents.
INSERT INTO store_documents (type, locale, bucket, object_key, content_type, size_bytes, filename)
SELECT 'terms', 'en', terms_bucket, terms_object_key, terms_content_type, terms_size_bytes, terms_filename
FROM store_settings WHERE terms_object_key IS NOT NULL;

INSERT INTO store_documents (type, locale, bucket, object_key, content_type, size_bytes, filename)
SELECT 'privacy', 'en', privacy_bucket, privacy_object_key, privacy_content_type, privacy_size_bytes, privacy_filename
FROM store_settings WHERE privacy_object_key IS NOT NULL;

ALTER TABLE store_settings
	DROP COLUMN terms_bucket,
	DROP COLUMN terms_object_key,
	DROP COLUMN terms_content_type,
	DROP COLUMN terms_size_bytes,
	DROP COLUMN terms_filename,
	DROP COLUMN privacy_bucket,
	DROP COLUMN privacy_object_key,
	DROP COLUMN privacy_content_type,
	DROP COLUMN privacy_size_bytes,
	DROP COLUMN privacy_filename;

-- +goose Down

ALTER TABLE store_settings
	ADD COLUMN terms_bucket TEXT,
	ADD COLUMN terms_object_key TEXT,
	ADD COLUMN terms_content_type TEXT,
	ADD COLUMN terms_size_bytes BIGINT,
	ADD COLUMN terms_filename TEXT,
	ADD COLUMN privacy_bucket TEXT,
	ADD COLUMN privacy_object_key TEXT,
	ADD COLUMN privacy_content_type TEXT,
	ADD COLUMN privacy_size_bytes BIGINT,
	ADD COLUMN privacy_filename TEXT;

UPDATE store_settings s SET
	terms_bucket = d.bucket, terms_object_key = d.object_key, terms_content_type = d.content_type,
	terms_size_bytes = d.size_bytes, terms_filename = d.filename
FROM store_documents d WHERE d.type = 'terms' AND d.locale = 'en';

UPDATE store_settings s SET
	privacy_bucket = d.bucket, privacy_object_key = d.object_key, privacy_content_type = d.content_type,
	privacy_size_bytes = d.size_bytes, privacy_filename = d.filename
FROM store_documents d WHERE d.type = 'privacy' AND d.locale = 'en';

DROP TABLE store_documents;
DROP TABLE ui_strings;
DROP TABLE translations;
DROP TABLE store_languages;

ALTER TABLE store_settings
	ADD COLUMN address_line1 TEXT,
	ADD COLUMN address_line2 TEXT,
	ADD COLUMN city TEXT,
	ADD COLUMN region TEXT,
	ADD COLUMN postal_code TEXT,
	ADD COLUMN country TEXT;

UPDATE store_settings s SET
	address_line1 = a.line1, address_line2 = a.line2, city = a.city,
	region = a.region, postal_code = a.postal_code, country = a.country
FROM store_addresses a WHERE a.store_settings_id = s.id AND a.is_default = true;

DROP TABLE store_addresses;

ALTER TABLE store_settings DROP COLUMN legal_entity_name;
