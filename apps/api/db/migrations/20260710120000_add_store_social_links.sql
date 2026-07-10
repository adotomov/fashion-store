-- +goose Up

-- Social media profile URLs shown in the storefront footer, editable under
-- Settings → Identity. Nullable — empty means the icon is hidden.
ALTER TABLE store_settings
    ADD COLUMN facebook_url  TEXT,
    ADD COLUMN instagram_url TEXT;

-- +goose Down

ALTER TABLE store_settings
    DROP COLUMN facebook_url,
    DROP COLUMN instagram_url;
