-- +migrate Up

-- Availabilities become per guild. Rows written before belong to the only guild that
-- could be configured until now; without any configured guild they can't be kept.
ALTER TABLE dispos ADD COLUMN IF NOT EXISTS guild_id TEXT;
UPDATE dispos SET guild_id = (SELECT guild_id FROM guild_config ORDER BY guild_id LIMIT 1) WHERE guild_id IS NULL;
DELETE FROM dispos WHERE guild_id IS NULL;
ALTER TABLE dispos ALTER COLUMN guild_id SET NOT NULL;
ALTER TABLE dispos DROP CONSTRAINT IF EXISTS dispos_unique_per_week;
-- Column order serves the lookups by (guild, week) and (guild, week, user).
ALTER TABLE dispos ADD CONSTRAINT dispos_unique_per_week UNIQUE (guild_id, week, user_id, day_index, slot_index);

ALTER TABLE week_unavailable ADD COLUMN IF NOT EXISTS guild_id TEXT;
UPDATE week_unavailable SET guild_id = (SELECT guild_id FROM guild_config ORDER BY guild_id LIMIT 1) WHERE guild_id IS NULL;
DELETE FROM week_unavailable WHERE guild_id IS NULL;
ALTER TABLE week_unavailable ALTER COLUMN guild_id SET NOT NULL;
ALTER TABLE week_unavailable DROP CONSTRAINT IF EXISTS week_unavailable_pk;
ALTER TABLE week_unavailable ADD CONSTRAINT week_unavailable_pk PRIMARY KEY (guild_id, week, user_id);

-- One weekly message per guild (the latest one), with its channel, instead of every
-- message ever posted. Single-guild versions could take the channel from the environment
-- only: the channel is then left empty and the bot uses the guild's configured channel.
CREATE TABLE IF NOT EXISTS weekly_messages (
    guild_id   TEXT PRIMARY KEY,
    week       TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    message_id TEXT NOT NULL
);
INSERT INTO weekly_messages (guild_id, week, channel_id, message_id)
SELECT g.guild_id, s.week, g.channel_id, s.message_id
FROM (SELECT week, message_id FROM sondage_messages ORDER BY week DESC LIMIT 1) s
CROSS JOIN (SELECT guild_id, COALESCE(channel_id, '') AS channel_id FROM guild_config ORDER BY guild_id LIMIT 1) g
ON CONFLICT (guild_id) DO NOTHING;
DROP TABLE IF EXISTS sondage_messages;
