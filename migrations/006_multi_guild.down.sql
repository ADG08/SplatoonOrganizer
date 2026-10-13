-- +migrate Down

CREATE TABLE IF NOT EXISTS sondage_messages (
    week       TEXT PRIMARY KEY,
    message_id TEXT NOT NULL
);
INSERT INTO sondage_messages (week, message_id)
SELECT week, message_id FROM weekly_messages
ON CONFLICT (week) DO NOTHING;
DROP TABLE IF EXISTS weekly_messages;

ALTER TABLE week_unavailable DROP CONSTRAINT IF EXISTS week_unavailable_pk;
DELETE FROM week_unavailable a USING week_unavailable b
WHERE a.ctid < b.ctid AND a.user_id = b.user_id AND a.week = b.week;
ALTER TABLE week_unavailable ADD CONSTRAINT week_unavailable_pk PRIMARY KEY (user_id, week);
ALTER TABLE week_unavailable DROP COLUMN IF EXISTS guild_id;

ALTER TABLE dispos DROP CONSTRAINT IF EXISTS dispos_unique_per_week;
DELETE FROM dispos a USING dispos b
WHERE a.ctid < b.ctid AND a.user_id = b.user_id AND a.day_index = b.day_index
  AND a.slot_index = b.slot_index AND a.week = b.week;
ALTER TABLE dispos ADD CONSTRAINT dispos_unique_per_week UNIQUE (user_id, day_index, slot_index, week);
ALTER TABLE dispos DROP COLUMN IF EXISTS guild_id;
