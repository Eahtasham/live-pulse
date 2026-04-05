CREATE TABLE IF NOT EXISTS polls (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id     UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    question       TEXT NOT NULL,
    answer_mode    TEXT NOT NULL DEFAULT 'single' CHECK (answer_mode IN ('single', 'multi')),
    time_limit_sec INTEGER,
    status         TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'closed')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_polls_session_id ON polls(session_id);
