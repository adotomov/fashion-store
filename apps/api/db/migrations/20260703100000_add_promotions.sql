-- +goose Up

CREATE TABLE promotions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL CHECK (type IN ('percentage', 'fixed', 'bxgy')),

    -- Percentage discount (type='percentage'): 1-100
    value_percent INTEGER,

    -- Fixed amount discount (type='fixed')
    value_fixed_minor BIGINT,
    value_fixed_currency TEXT DEFAULT 'EUR',

    -- Buy X Get Y (type='bxgy')
    buy_qty INTEGER,
    get_qty INTEGER,
    get_discount_pct INTEGER, -- 100=free, 50=50% off the "get" items

    -- Minimum cart quantity before promo triggers (all types)
    min_quantity INTEGER NOT NULL DEFAULT 1,

    -- Scope: what products does this apply to?
    target_type TEXT NOT NULL DEFAULT 'all' CHECK (target_type IN ('all', 'category', 'product_type', 'product')),

    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    priority INTEGER NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE promotion_categories (
    promotion_id UUID NOT NULL REFERENCES promotions (id) ON DELETE CASCADE,
    category_id  UUID NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    PRIMARY KEY (promotion_id, category_id)
);

CREATE TABLE promotion_product_types (
    promotion_id    UUID NOT NULL REFERENCES promotions (id) ON DELETE CASCADE,
    product_type_id UUID NOT NULL REFERENCES product_types (id) ON DELETE CASCADE,
    PRIMARY KEY (promotion_id, product_type_id)
);

CREATE TABLE promotion_products (
    promotion_id UUID NOT NULL REFERENCES promotions (id) ON DELETE CASCADE,
    product_id   UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    PRIMARY KEY (promotion_id, product_id)
);

CREATE TABLE discount_codes (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code          TEXT NOT NULL UNIQUE,
    value_percent INTEGER NOT NULL CHECK (value_percent BETWEEN 1 AND 100),
    starts_at     TIMESTAMPTZ,
    expires_at    TIMESTAMPTZ,
    max_uses      INTEGER,
    use_count     INTEGER NOT NULL DEFAULT 0,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX discount_codes_code_idx ON discount_codes (code);

ALTER TABLE orders
    ADD COLUMN discount_code TEXT,
    ADD COLUMN discount_amount_minor BIGINT,
    ADD COLUMN discount_amount_currency TEXT;

-- +goose Down
ALTER TABLE orders
    DROP COLUMN discount_code,
    DROP COLUMN discount_amount_minor,
    DROP COLUMN discount_amount_currency;

DROP INDEX discount_codes_code_idx;
DROP TABLE discount_codes;
DROP TABLE promotion_products;
DROP TABLE promotion_product_types;
DROP TABLE promotion_categories;
DROP TABLE promotions;
