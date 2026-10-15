-- +migrate Up

-- How the weekly availability message works, configured from /help.
-- slot_mode: NULL or "halfday" = Après-midi / Soir, "hourly" = one slot per hour
-- between slot_start_hour and slot_end_hour (24 = minuit), in Paris time.
-- display_style: NULL or "classic" = embed with a text table, "image" = grid image.
ALTER TABLE guild_config ADD COLUMN IF NOT EXISTS slot_mode       TEXT;
ALTER TABLE guild_config ADD COLUMN IF NOT EXISTS slot_start_hour SMALLINT;
ALTER TABLE guild_config ADD COLUMN IF NOT EXISTS slot_end_hour   SMALLINT;
ALTER TABLE guild_config ADD COLUMN IF NOT EXISTS display_style   TEXT;
