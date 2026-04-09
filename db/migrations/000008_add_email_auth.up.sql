-- Add password_hash column and allow 'email' as a provider
ALTER TABLE users ADD COLUMN password_hash TEXT;
ALTER TABLE users DROP CONSTRAINT users_provider_check;
ALTER TABLE users ADD CONSTRAINT users_provider_check CHECK (provider IN ('google', 'magic-link', 'email'));
