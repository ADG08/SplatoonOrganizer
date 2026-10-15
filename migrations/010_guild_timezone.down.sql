-- +migrate Down

ALTER TABLE guild_config DROP COLUMN IF EXISTS timezone;
