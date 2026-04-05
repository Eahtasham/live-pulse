CREATE TABLE IF NOT EXISTS poll_options (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poll_id  UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    label    TEXT NOT NULL,
    position SMALLINT NOT NULL
);

CREATE INDEX idx_poll_options_poll_id ON poll_options(poll_id);
