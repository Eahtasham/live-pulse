CREATE TABLE IF NOT EXISTS qa_entries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    author_uid  TEXT NOT NULL,
    entry_type  TEXT NOT NULL CHECK (entry_type IN ('question', 'comment')),
    body        TEXT NOT NULL,
    score       INTEGER NOT NULL DEFAULT 0,
    status      TEXT NOT NULL DEFAULT 'visible' CHECK (status IN ('visible', 'answered', 'pinned', 'archived')),
    is_hidden   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_qa_entries_session_id ON qa_entries(session_id);
CREATE INDEX idx_qa_entries_session_status ON qa_entries(session_id, status);
