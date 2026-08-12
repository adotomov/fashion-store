-- +goose Up
-- Preferred display/email language for the account, captured from the Google
-- account's `locale` claim at sign-in and normalised to a primary language
-- subtag (e.g. "en", "bg"). Empty means "no preference recorded" — the store
-- falls back to geo/browser detection for the site and to the store locale for
-- email. Kept on the user (not just the session) so transactional email can be
-- sent in the recipient's own language.
ALTER TABLE users ADD COLUMN locale TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE users DROP COLUMN locale;
