-- +migrate Up

-- Whether members see the hourly slots in their own timezone (asked on their first
-- click), for international guilds. FALSE: everyone uses the guild's timezone.
ALTER TABLE guild_config ADD COLUMN IF NOT EXISTS member_timezones BOOLEAN NOT NULL DEFAULT FALSE;
