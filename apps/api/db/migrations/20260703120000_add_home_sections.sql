-- +goose Up

CREATE TABLE home_sections (
    id         TEXT PRIMARY KEY,
    enabled    BOOLEAN NOT NULL DEFAULT FALSE,
    eyebrow    TEXT NOT NULL DEFAULT '',
    heading    TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE home_section_products (
    section_id TEXT NOT NULL REFERENCES home_sections(id) ON DELETE CASCADE,
    product_id UUID NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    PRIMARY KEY (section_id, product_id)
);

INSERT INTO home_sections (id, enabled, eyebrow, heading) VALUES
    ('spotlights',       FALSE, 'Editor''s Choice', 'Spotlights'),
    ('recommended',      FALSE, 'Staff Picks',      'Recommended by Us'),
    ('on_sale',          FALSE, 'Limited Time',      'What''s on Sale'),
    ('best_in_category', FALSE, 'Top Picks',         'Best in its Category');

-- +goose Down
DROP TABLE home_section_products;
DROP TABLE home_sections;
