-- +goose Up

-- Make GCS storage columns nullable so a row can hold either a file
-- reference (legacy PDF upload) or inline Markdown text.
ALTER TABLE store_documents
    ADD COLUMN content_md TEXT,
    ALTER COLUMN bucket       DROP NOT NULL,
    ALTER COLUMN object_key   DROP NOT NULL,
    ALTER COLUMN content_type DROP NOT NULL,
    ALTER COLUMN size_bytes   DROP NOT NULL,
    ALTER COLUMN filename     DROP NOT NULL;

-- +goose Down

-- Clear content_md rows before re-adding NOT NULL constraints.
UPDATE store_documents SET bucket = '', object_key = '', content_type = '', size_bytes = 0, filename = ''
WHERE bucket IS NULL;

ALTER TABLE store_documents
    DROP COLUMN content_md,
    ALTER COLUMN bucket       SET NOT NULL,
    ALTER COLUMN object_key   SET NOT NULL,
    ALTER COLUMN content_type SET NOT NULL,
    ALTER COLUMN size_bytes   SET NOT NULL,
    ALTER COLUMN filename     SET NOT NULL;
