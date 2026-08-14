-- +goose Up
-- The display language the customer was using when they placed the order,
-- normalised to a primary language subtag (e.g. "en", "bg"). Captured from the
-- storefront so transactional email can be sent in the customer's own language
-- even for guests, who have no user record to read a preference from. Empty
-- means "not captured" — email then falls back to the store default locale.
-- A signed-in user's saved preference still wins over this (account-wins).
ALTER TABLE orders ADD COLUMN locale TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE orders DROP COLUMN locale;
