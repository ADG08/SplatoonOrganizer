-- +migrate Down

-- A guild goes back to one configuration: its oldest team's, with that team's
-- availabilities and weekly message. Its other teams are lost.
DROP TABLE IF EXISTS guilds;

CREATE TABLE IF NOT EXISTS guild_config (
    guild_id         TEXT PRIMARY KEY,
    channel_id       TEXT,
    role_id          TEXT,
    slot_mode        TEXT,
    slot_start_hour  SMALLINT,
    slot_end_hour    SMALLINT,
    display_style    TEXT,
    language         TEXT,
    publish_weekday  SMALLINT,
    publish_hour     SMALLINT,
    active           BOOLEAN NOT NULL DEFAULT TRUE,
    timezone         TEXT,
    member_timezones BOOLEAN NOT NULL DEFAULT FALSE,
    full_at          SMALLINT
);

INSERT INTO guild_config (guild_id, channel_id, role_id, slot_mode, slot_start_hour, slot_end_hour,
                          display_style, language, publish_weekday, publish_hour, active, timezone,
                          member_timezones, full_at)
SELECT DISTINCT ON (guild_id)
       guild_id, channel_id, role_id, slot_mode, slot_start_hour, slot_end_hour, display_style,
       language, publish_weekday, publish_hour, active, timezone, member_timezones, full_at
FROM teams
ORDER BY guild_id, id;

ALTER TABLE dispos ADD COLUMN IF NOT EXISTS guild_id TEXT;
UPDATE dispos d SET guild_id = t.guild_id FROM teams t WHERE t.id = d.team_id;
DELETE FROM dispos d WHERE d.team_id <> (SELECT MIN(id) FROM teams t WHERE t.guild_id = d.guild_id);
ALTER TABLE dispos ALTER COLUMN guild_id SET NOT NULL;
ALTER TABLE dispos DROP CONSTRAINT IF EXISTS dispos_unique_per_week;
ALTER TABLE dispos DROP COLUMN IF EXISTS team_id;
ALTER TABLE dispos ADD CONSTRAINT dispos_unique_per_week UNIQUE (guild_id, week, user_id, day_index, slot_index);

ALTER TABLE week_unavailable ADD COLUMN IF NOT EXISTS guild_id TEXT;
UPDATE week_unavailable w SET guild_id = t.guild_id FROM teams t WHERE t.id = w.team_id;
DELETE FROM week_unavailable w WHERE w.team_id <> (SELECT MIN(id) FROM teams t WHERE t.guild_id = w.guild_id);
ALTER TABLE week_unavailable ALTER COLUMN guild_id SET NOT NULL;
ALTER TABLE week_unavailable DROP CONSTRAINT IF EXISTS week_unavailable_pk;
ALTER TABLE week_unavailable DROP COLUMN IF EXISTS team_id;
ALTER TABLE week_unavailable ADD CONSTRAINT week_unavailable_pk PRIMARY KEY (guild_id, week, user_id);

ALTER TABLE weekly_messages ADD COLUMN IF NOT EXISTS guild_id TEXT;
UPDATE weekly_messages m SET guild_id = t.guild_id FROM teams t WHERE t.id = m.team_id;
DELETE FROM weekly_messages m WHERE m.team_id <> (SELECT MIN(id) FROM teams t WHERE t.guild_id = m.guild_id);
ALTER TABLE weekly_messages ALTER COLUMN guild_id SET NOT NULL;
ALTER TABLE weekly_messages DROP COLUMN IF EXISTS team_id;
ALTER TABLE weekly_messages ADD CONSTRAINT weekly_messages_pkey PRIMARY KEY (guild_id);

DROP TABLE IF EXISTS teams;
