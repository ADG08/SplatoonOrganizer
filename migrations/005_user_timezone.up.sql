-- +migrate Up

-- Timezone chosen by each member, used to show the hourly slots in their local time.
CREATE TABLE IF NOT EXISTS user_timezone (
    user_id    TEXT PRIMARY KEY,
    timezone   TEXT NOT NULL, -- IANA name, e.g. "America/New_York"
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
