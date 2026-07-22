-- +goose Up
-- Singleton editorial ("Shop the Look") banner shown mid-home-page. Same
-- single-row shape as hero_settings: the row is seeded here and only ever
-- upserted. Disabled by default so nothing renders until an admin configures it.
CREATE TABLE editorial_banner_settings (
    id                   BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
    enabled              BOOLEAN NOT NULL DEFAULT FALSE,
    eyebrow              TEXT NOT NULL DEFAULT 'The Edit',
    heading              TEXT NOT NULL DEFAULT 'Shop the look',
    subtext              TEXT NOT NULL DEFAULT 'Discover our latest styling, curated to wear together.',
    cta_label            TEXT NOT NULL DEFAULT 'Explore the edit',
    cta_url              TEXT NOT NULL DEFAULT '/shop',
    image_bucket         TEXT,
    image_object_key     TEXT,
    image_content_type   TEXT,
    image_size_bytes     BIGINT,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO editorial_banner_settings DEFAULT VALUES;

-- +goose Down
DROP TABLE editorial_banner_settings;
