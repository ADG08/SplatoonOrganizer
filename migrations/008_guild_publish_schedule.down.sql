-- +migrate Down

ALTER TABLE guild_config DROP COLUMN IF EXISTS publish_hour;
ALTER TABLE guild_config DROP COLUMN IF EXISTS publish_weekday;
