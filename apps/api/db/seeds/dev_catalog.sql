-- Development catalog seed.
--
-- NOT a migration — this is demo data and must never run in production. Apply
-- it with `make seed-dev-catalog` (devbox) or by piping it into psql.
--
-- Adds the "Women's Clothing" product type, ten new categories, and ten active
-- products (with variants + stock) in every category that exists afterwards,
-- including the ones already present. Safe to re-run: every insert is keyed on
-- a unique slug and skipped if it is already there.

BEGIN;

-- The Bracelets category was created with the slug 'dresses' by hand, which
-- collides with the Women's Clothing category added below.
UPDATE categories SET slug = 'bracelets' WHERE name = 'Bracelets' AND slug = 'dresses';

INSERT INTO product_types (name, slug, position)
VALUES ('Women''s Clothing', 'womens-clothing', 1)
ON CONFLICT (slug) DO NOTHING;

INSERT INTO categories (name, slug, product_type_id)
SELECT v.name, v.slug, t.id
FROM (VALUES
    ('Earrings',  'earrings',  'accessories'),
    ('Necklaces', 'necklaces', 'accessories'),
    ('Blouses',   'blouses',   'womens-clothing'),
    ('Dresses',   'dresses',   'womens-clothing'),
    ('Jeans',     'jeans',     'womens-clothing'),
    ('Overalls',  'overalls',  'womens-clothing'),
    ('Coats',     'coats',     'womens-clothing'),
    ('Jackets',   'jackets',   'womens-clothing'),
    ('Pants',     'pants',     'womens-clothing'),
    ('Skirts',    'skirts',    'womens-clothing')
) AS v (name, slug, type_slug)
JOIN product_types t ON t.slug = v.type_slug
ON CONFLICT (slug) DO NOTHING;

DO $seed$
DECLARE
    -- Product names are "<noun> <style>", e.g. "Blouse Positano", which keeps
    -- both the name and the derived slug unique across the catalog.
    v_styles TEXT[] := ARRAY['Amalfi', 'Riviera', 'Capri', 'Verona', 'Sienna',
                             'Positano', 'Lucca', 'Ravello', 'Portofino', 'Bellagio'];
    v_size_codes TEXT[] := ARRAY['S', 'M', 'L'];
    -- Per category: singular noun, base price in minor units, and whether it
    -- is sized (clothing gets S/M/L variants, accessories get a single one).
    v_cats CONSTANT JSONB := '[
        {"slug": "bags",      "noun": "Bag",      "price":  8900, "sized": false},
        {"slug": "hats",      "noun": "Hat",      "price":  4500, "sized": false},
        {"slug": "bracelets", "noun": "Bracelet", "price":  5000, "sized": false},
        {"slug": "earrings",  "noun": "Earrings", "price":  3900, "sized": false},
        {"slug": "necklaces", "noun": "Necklace", "price":  5900, "sized": false},
        {"slug": "blouses",   "noun": "Blouse",   "price":  6900, "sized": true},
        {"slug": "dresses",   "noun": "Dress",    "price":  9900, "sized": true},
        {"slug": "jeans",     "noun": "Jeans",    "price":  7900, "sized": true},
        {"slug": "overalls",  "noun": "Overall",  "price":  8900, "sized": true},
        {"slug": "coats",     "noun": "Coat",     "price": 19900, "sized": true},
        {"slug": "jackets",   "noun": "Jacket",   "price": 14900, "sized": true},
        {"slug": "pants",     "noun": "Pants",    "price":  6900, "sized": true},
        {"slug": "skirts",    "noun": "Skirt",    "price":  5900, "sized": true}
    ]'::JSONB;

    v_cat           JSONB;
    v_category_id   UUID;
    v_color_attr_id UUID;
    v_size_attr_id  UUID;
    v_color_ids     UUID[];
    v_size_ids      UUID[];
    v_i             INT;
    v_s             INT;
    v_name          TEXT;
    v_slug          TEXT;
    v_product_id    UUID;
    v_variant_id    UUID;
    v_price         BIGINT;
    v_compare_at    BIGINT;
    v_color_id      UUID;
    v_sku_prefix    TEXT;
BEGIN
    SELECT id INTO v_color_attr_id FROM attributes WHERE lower(name) = 'color';
    SELECT id INTO v_size_attr_id FROM attributes WHERE lower(name) = 'size';

    SELECT array_agg(av.id ORDER BY pos.ord) INTO v_color_ids
    FROM unnest(ARRAY['Black', 'Clay', 'Blue', 'Green']) WITH ORDINALITY AS pos(value, ord)
    JOIN attribute_values av ON av.attribute_id = v_color_attr_id AND lower(av.value) = lower(pos.value);

    SELECT array_agg(av.id ORDER BY pos.ord) INTO v_size_ids
    FROM unnest(v_size_codes) WITH ORDINALITY AS pos(value, ord)
    JOIN attribute_values av ON av.attribute_id = v_size_attr_id AND lower(av.value) = lower(pos.value);

    IF v_color_ids IS NULL OR array_length(v_color_ids, 1) < 4
       OR v_size_ids IS NULL OR array_length(v_size_ids, 1) < 3 THEN
        RAISE EXCEPTION 'seed expects Color (Black/Clay/Blue/Green) and Size (S/M/L) attribute values to exist';
    END IF;

    FOR v_cat IN SELECT * FROM jsonb_array_elements(v_cats) LOOP
        SELECT id INTO v_category_id FROM categories WHERE slug = v_cat->>'slug';
        CONTINUE WHEN v_category_id IS NULL;

        FOR v_i IN 1..array_length(v_styles, 1) LOOP
            v_name := (v_cat->>'noun') || ' ' || v_styles[v_i];
            v_slug := lower(replace(v_name, ' ', '-'));
            v_price := (v_cat->>'price')::BIGINT + (v_i - 1) * 500;
            -- Every fourth product carries a struck-through original price so
            -- the sale styling has something to render.
            v_compare_at := CASE WHEN v_i % 4 = 0 THEN v_price + 2000 END;

            INSERT INTO products (name, slug, description, status,
                                  base_price_amount, base_price_currency,
                                  compare_at_price_amount, compare_at_price_currency)
            VALUES (v_name, v_slug,
                    v_name || ' — part of the ' || (v_cat->>'noun') ||
                        ' line, made in limited quantities.',
                    'active', v_price, 'EUR',
                    v_compare_at, CASE WHEN v_compare_at IS NULL THEN NULL ELSE 'EUR' END)
            ON CONFLICT (slug) DO NOTHING
            RETURNING id INTO v_product_id;

            -- Already seeded by an earlier run.
            CONTINUE WHEN v_product_id IS NULL;

            INSERT INTO product_categories (product_id, category_id) VALUES (v_product_id, v_category_id);

            v_color_id := v_color_ids[1 + (v_i - 1) % array_length(v_color_ids, 1)];
            v_sku_prefix := upper(regexp_replace(v_slug, '[^a-z0-9]', '', 'g'));

            INSERT INTO product_attributes (product_id, attribute_id) VALUES (v_product_id, v_color_attr_id);

            IF (v_cat->>'sized')::BOOLEAN THEN
                INSERT INTO product_attributes (product_id, attribute_id) VALUES (v_product_id, v_size_attr_id);

                FOR v_s IN 1..array_length(v_size_codes, 1) LOOP
                    INSERT INTO product_variants (product_id) VALUES (v_product_id) RETURNING id INTO v_variant_id;
                    INSERT INTO variant_attribute_values (variant_id, attribute_value_id)
                    VALUES (v_variant_id, v_color_id), (v_variant_id, v_size_ids[v_s]);
                    INSERT INTO inventory_items (variant_id, sku, quantity_on_hand)
                    VALUES (v_variant_id, v_sku_prefix || '-' || v_size_codes[v_s], 8);
                END LOOP;
            ELSE
                INSERT INTO product_variants (product_id) VALUES (v_product_id) RETURNING id INTO v_variant_id;
                INSERT INTO variant_attribute_values (variant_id, attribute_value_id)
                VALUES (v_variant_id, v_color_id);
                INSERT INTO inventory_items (variant_id, sku, quantity_on_hand)
                VALUES (v_variant_id, v_sku_prefix, 8);
            END IF;
        END LOOP;
    END LOOP;
END
$seed$;

COMMIT;
