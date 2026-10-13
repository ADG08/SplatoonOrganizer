-- +migrate Up

-- Language of the bot in the guild, set from /help. NULL means French.
ALTER TABLE guild_config ADD COLUMN IF NOT EXISTS language TEXT;
