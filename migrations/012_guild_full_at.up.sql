-- +migrate Up

-- Players from which a slot is shown full (green) in the grid image, set from /help.
-- NULL: relative to the week's best slot.
ALTER TABLE guild_config ADD COLUMN IF NOT EXISTS full_at SMALLINT;
