CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "to" VARCHAR(32) NOT NULL,
    content VARCHAR(160) NOT NULL,
    sent BOOLEAN NOT NULL DEFAULT false,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messages_sent_created_at ON messages (sent, created_at);
CREATE INDEX IF NOT EXISTS idx_messages_sent_at ON messages (sent_at DESC);
