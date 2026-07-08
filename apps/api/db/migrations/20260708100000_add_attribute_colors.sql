-- +goose Up
-- Attributes gain a type (plain text vs. a color swatch) and a system flag.
-- System attributes are the built-in "Default" set (currently just Color)
-- that ship with the store and can't be deleted by admins.
ALTER TABLE attributes ADD COLUMN type TEXT NOT NULL DEFAULT 'text';
ALTER TABLE attributes ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE attributes ADD CONSTRAINT attributes_type_check CHECK (type IN ('text', 'color'));

-- Color-typed attribute values carry the picked palette color alongside the
-- value's name. NULL for plain-text attributes.
ALTER TABLE attribute_values ADD COLUMN color_hex TEXT;

-- Adopt an existing "Color" attribute (an admin may already have created one)
-- as the built-in color attribute, upgrading it to the color type so its
-- values render as swatches.
UPDATE attributes SET type = 'color', is_system = true WHERE lower(name) = 'color';

-- Otherwise seed the built-in Color attribute from scratch.
INSERT INTO attributes (name, type, is_system)
SELECT 'Color', 'color', true
WHERE NOT EXISTS (SELECT 1 FROM attributes WHERE lower(name) = 'color');

-- +goose Down
-- Only remove a seeded Color attribute we created (one with no values); an
-- adopted, pre-existing Color attribute is left in place.
DELETE FROM attributes a
WHERE a.is_system = true AND a.type = 'color' AND lower(a.name) = 'color'
	AND NOT EXISTS (SELECT 1 FROM attribute_values v WHERE v.attribute_id = a.id);

ALTER TABLE attribute_values DROP COLUMN color_hex;

ALTER TABLE attributes DROP CONSTRAINT attributes_type_check;
ALTER TABLE attributes DROP COLUMN is_system;
ALTER TABLE attributes DROP COLUMN type;
