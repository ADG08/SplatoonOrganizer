-- +migrate Up

-- Whether the weekly message is published in the guild. Turned off when the bot leaves the
-- guild or loses access to its channel, back on when it returns or a channel is set again:
-- the configuration is kept meanwhile.
ALTER TABLE guild_config ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT TRUE;
