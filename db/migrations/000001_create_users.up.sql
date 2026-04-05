CREATE TABLE IF NOT EXISTS users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT UNIQUE NOT NULL,
    name       TEXT,
    avatar_url TEXT,
    provider   TEXT NOT NULL CHECK (provider IN ('google', 'magic-link')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
