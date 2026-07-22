-- +goose Up
-- email_messages is the transactional outbox for every outbound email. Producers
-- (checkout, auth, fulfillment) never call the email provider inline — they
-- enqueue a row here inside their own transaction, and a background dispatcher
-- claims, renders and sends it. That keeps the request path independent of the
-- provider's availability and guarantees an order confirmation survives a
-- provider outage, since an unsent row is simply retried.
--
-- Claiming uses FOR UPDATE SKIP LOCKED plus a lease: the dispatcher flips a due
-- row to 'sending' and pushes next_attempt_at forward, so a crash mid-send
-- leaves a row that naturally becomes due again rather than one stuck forever.
CREATE TABLE email_messages (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	template_key TEXT NOT NULL,
	locale TEXT NOT NULL,
	to_email TEXT NOT NULL,
	to_name TEXT NOT NULL DEFAULT '',
	-- Template variables, merged with store branding at render time.
	payload JSONB NOT NULL DEFAULT '{}'::jsonb,
	-- Makes enqueue idempotent: a retried producer or a duplicate provider event
	-- can never queue the same message twice (e.g. order_confirmation:<order_id>).
	dedupe_key TEXT NOT NULL UNIQUE,
	-- Nullable classification, used only for metrics/filtering today. This is the
	-- seam a future marketing stream would use; transactional mail leaves it NULL.
	category TEXT,
	status TEXT NOT NULL DEFAULT 'pending'
		CHECK (status IN ('pending', 'sending', 'sent', 'failed', 'suppressed')),
	attempts INT NOT NULL DEFAULT 0,
	-- When this row next becomes eligible for a send attempt. Doubles as the
	-- lease expiry while status = 'sending'.
	next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	last_error TEXT,
	provider_message_id TEXT,
	sent_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Drives the dispatcher's claim query; only rows still in flight are indexed so
-- the index stays small as sent history accumulates.
CREATE INDEX email_messages_due_idx ON email_messages (next_attempt_at)
	WHERE status IN ('pending', 'sending');

-- +goose Down
DROP INDEX IF EXISTS email_messages_due_idx;
DROP TABLE email_messages;
