-- +migrate Down

ALTER TABLE guild_config DROP COLUMN IF EXISTS display_style;
ALTER TABLE guild_config DROP COLUMN IF EXISTS slot_end_hour;
ALTER TABLE guild_config DROP COLUMN IF EXISTS slot_start_hour;
ALTER TABLE guild_config DROP COLUMN IF EXISTS slot_mode;
