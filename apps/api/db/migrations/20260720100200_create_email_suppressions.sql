-- +goose Up
-- email_suppressions is the list of addresses we must never mail again. It is
-- fed by the provider's event webhook: a hard bounce means the mailbox does not
-- exist, and a spam complaint means the recipient asked us to stop. Continuing
-- to send to either is what destroys a domain's sending reputation, which is
-- why this applies to transactional mail too, not just marketing.
--
-- The dispatcher checks this immediately before every send, so an address
-- suppressed after a message was queued still stops that message.
CREATE TABLE email_suppressions (
	email TEXT PRIMARY KEY,
	-- 'bounce' (permanent delivery failure) or 'complaint' (marked as spam).
	reason TEXT NOT NULL CHECK (reason IN ('bounce', 'complaint')),
	detail TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE email_suppressions;
