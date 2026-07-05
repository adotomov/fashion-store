-- +goose Up
CREATE TABLE hero_settings (
    id                   BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
    eyebrow              TEXT NOT NULL DEFAULT 'New Season',
    heading              TEXT NOT NULL DEFAULT 'Quietly considered style, for every day.',
    subtext              TEXT NOT NULL DEFAULT 'Clothing, jewelry, bags, and accessories — thoughtfully made, finished by hand, and built to last beyond a single season.',
    cta_primary_label    TEXT NOT NULL DEFAULT 'Shop All Items',
    cta_primary_url      TEXT NOT NULL DEFAULT '/shop',
    cta_secondary_label  TEXT,
    cta_secondary_url    TEXT,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO hero_settings DEFAULT VALUES;

-- +goose Down
DROP TABLE hero_settings;
