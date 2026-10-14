-- +migrate Up

-- Timezone of the guild's hours (slots, grid, weeks, publication), set from /help:
-- an IANA name ("Europe/Paris") or a fixed offset ("UTC+2"). NULL means Europe/Paris.
ALTER TABLE guild_config ADD COLUMN IF NOT EXISTS timezone TEXT;
