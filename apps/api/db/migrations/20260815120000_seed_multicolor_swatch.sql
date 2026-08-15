-- +goose Up
-- Seed a permanent, built-in "Multicolor" swatch on the system Color attribute
-- for multi-colored garments (prints, patterns). It carries the sentinel
-- color_hex 'multicolor' (deliberately not a valid hex) so the storefront
-- renders it as a rainbow wheel and the admin API refuses to delete it. Idempotent.
INSERT INTO attribute_values (attribute_id, value, color_hex)
SELECT a.id, 'Multicolor', 'multicolor'
FROM attributes a
WHERE a.is_system = true
	AND a.type = 'color'
	AND lower(a.name) = 'color'
	AND NOT EXISTS (
		SELECT 1 FROM attribute_values v
		WHERE v.attribute_id = a.id AND v.color_hex = 'multicolor'
	);

-- +goose Down
DELETE FROM attribute_values WHERE color_hex = 'multicolor';
