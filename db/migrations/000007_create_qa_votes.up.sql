CREATE TABLE IF NOT EXISTS qa_votes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    qa_entry_id UUID NOT NULL REFERENCES qa_entries(id) ON DELETE CASCADE,
    voter_uid   TEXT NOT NULL,
    vote_value  SMALLINT NOT NULL CHECK (vote_value IN (-1, 1)),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (qa_entry_id, voter_uid)
);

CREATE INDEX idx_qa_votes_qa_entry_id ON qa_votes(qa_entry_id);
