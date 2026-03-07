-- +migrate Up

CREATE TABLE IF NOT EXISTS guild_config (
    guild_id   TEXT PRIMARY KEY,
    channel_id TEXT,
    role_id    TEXT
);
