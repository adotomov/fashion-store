-- +goose Up

-- The "Best in its category" home section is curated as up to 5 categories,
-- each with its own hand-picked, ordered list of products. Two tables: one for
-- the category selection + order, one for the per-category product picks.

CREATE TABLE home_section_categories (
    section_id  TEXT NOT NULL REFERENCES home_sections(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    sort_order  INT NOT NULL DEFAULT 0,
    PRIMARY KEY (section_id, category_id)
);

CREATE TABLE home_section_category_products (
    section_id  TEXT NOT NULL,
    category_id UUID NOT NULL,
    product_id  UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    sort_order  INT NOT NULL DEFAULT 0,
    PRIMARY KEY (section_id, category_id, product_id),
    -- Picks disappear automatically when their category is removed from the
    -- section (or the section itself is deleted).
    FOREIGN KEY (section_id, category_id)
        REFERENCES home_section_categories(section_id, category_id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE home_section_category_products;
DROP TABLE home_section_categories;
