-- +migrate Up

-- When the weekly message is published, set from /help. NULL means Monday 10h.
-- publish_weekday: 0 = Sunday (for the week that follows), 1 = Monday.
ALTER TABLE guild_config ADD COLUMN IF NOT EXISTS publish_weekday SMALLINT;
ALTER TABLE guild_config ADD COLUMN IF NOT EXISTS publish_hour SMALLINT;
