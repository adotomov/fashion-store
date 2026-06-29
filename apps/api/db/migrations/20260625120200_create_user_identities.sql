-- +goose Up
CREATE TABLE user_identities (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	provider TEXT NOT NULL,
	provider_subject TEXT NOT NULL,
	email TEXT NOT NULL,
	email_verified BOOLEAN NOT NULL DEFAULT FALSE,
	raw_profile JSONB,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX user_identities_provider_subject_idx ON user_identities (provider, provider_subject);
CREATE INDEX user_identities_user_id_idx ON user_identities (user_id);

-- +goose Down
DROP TABLE user_identities;
