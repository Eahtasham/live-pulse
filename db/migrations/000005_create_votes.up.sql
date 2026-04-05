CREATE TABLE IF NOT EXISTS votes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poll_id      UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    option_id    UUID NOT NULL REFERENCES poll_options(id) ON DELETE CASCADE,
    audience_uid TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (poll_id, audience_uid, option_id)
);

CREATE INDEX idx_votes_poll_id ON votes(poll_id);
CREATE INDEX idx_votes_option_id ON votes(option_id);
