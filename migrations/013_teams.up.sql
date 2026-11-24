-- +migrate Up

-- A guild now holds several teams, each with its own settings, weekly message and
-- availabilities: guild_config's columns move to teams. An existing guild keeps everything
-- in one team, named in its language.
CREATE TABLE IF NOT EXISTS teams (
    id               BIGSERIAL PRIMARY KEY,
    guild_id         TEXT NOT NULL,
    name             TEXT NOT NULL,
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
    full_at          SMALLINT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT teams_name_per_guild UNIQUE (guild_id, name)
);

INSERT INTO teams (guild_id, name, channel_id, role_id, slot_mode, slot_start_hour, slot_end_hour,
                   display_style, language, publish_weekday, publish_hour, active, timezone,
                   member_timezones, full_at)
SELECT guild_id, CASE WHEN language = 'en' THEN 'Main team' ELSE 'Équipe principale' END,
       channel_id, role_id, slot_mode, slot_start_hour, slot_end_hour, display_style, language,
       publish_weekday, publish_hour, active, timezone, member_timezones, full_at
FROM guild_config;

DROP TABLE IF EXISTS guild_config;

-- Every guild has one team at this point. Rows of a guild without configuration can't be
-- kept: nothing could show them.
ALTER TABLE dispos ADD COLUMN IF NOT EXISTS team_id BIGINT REFERENCES teams (id) ON DELETE CASCADE;
UPDATE dispos d SET team_id = t.id FROM teams t WHERE t.guild_id = d.guild_id;
DELETE FROM dispos WHERE team_id IS NULL;
ALTER TABLE dispos ALTER COLUMN team_id SET NOT NULL;
ALTER TABLE dispos DROP CONSTRAINT IF EXISTS dispos_unique_per_week;
ALTER TABLE dispos DROP COLUMN IF EXISTS guild_id;
-- Column order serves the lookups by (team, week) and (team, week, user).
ALTER TABLE dispos ADD CONSTRAINT dispos_unique_per_week UNIQUE (team_id, week, user_id, day_index, slot_index);

ALTER TABLE week_unavailable ADD COLUMN IF NOT EXISTS team_id BIGINT REFERENCES teams (id) ON DELETE CASCADE;
UPDATE week_unavailable w SET team_id = t.id FROM teams t WHERE t.guild_id = w.guild_id;
DELETE FROM week_unavailable WHERE team_id IS NULL;
ALTER TABLE week_unavailable ALTER COLUMN team_id SET NOT NULL;
ALTER TABLE week_unavailable DROP CONSTRAINT IF EXISTS week_unavailable_pk;
ALTER TABLE week_unavailable DROP COLUMN IF EXISTS guild_id;
ALTER TABLE week_unavailable ADD CONSTRAINT week_unavailable_pk PRIMARY KEY (team_id, week, user_id);

ALTER TABLE weekly_messages ADD COLUMN IF NOT EXISTS team_id BIGINT REFERENCES teams (id) ON DELETE CASCADE;
UPDATE weekly_messages m SET team_id = t.id FROM teams t WHERE t.guild_id = m.guild_id;
DELETE FROM weekly_messages WHERE team_id IS NULL;
ALTER TABLE weekly_messages ALTER COLUMN team_id SET NOT NULL;
ALTER TABLE weekly_messages DROP COLUMN IF EXISTS guild_id;
ALTER TABLE weekly_messages ADD CONSTRAINT weekly_messages_pkey PRIMARY KEY (team_id);

-- What concerns a guild as a whole: the language picked for what isn't a team's (the list
-- of its teams, the welcome message, its new teams), NULL until picked, and the message
-- posted when the bot joined it, updated as its setup goes on.
CREATE TABLE IF NOT EXISTS guilds (
    guild_id           TEXT PRIMARY KEY,
    language           TEXT,
    welcome_channel_id TEXT,
    welcome_message_id TEXT
);
