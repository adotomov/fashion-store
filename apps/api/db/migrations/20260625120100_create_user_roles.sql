-- +goose Up
CREATE TABLE user_roles (
	user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	role TEXT NOT NULL CHECK (role IN ('user', 'admin')),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (user_id, role)
);

-- +goose Down
DROP TABLE user_roles;
