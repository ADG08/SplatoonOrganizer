-- name: GetGuild :one
SELECT *
FROM guilds
WHERE guild_id = $1;

-- name: SetGuildLanguage :exec
INSERT INTO guilds (guild_id, language)
VALUES ($1, $2)
ON CONFLICT (guild_id) DO UPDATE
SET language = EXCLUDED.language;

-- name: SetGuildWelcome :exec
INSERT INTO guilds (guild_id, welcome_channel_id, welcome_message_id)
VALUES ($1, $2, $3)
ON CONFLICT (guild_id) DO UPDATE
SET welcome_channel_id = EXCLUDED.welcome_channel_id,
    welcome_message_id = EXCLUDED.welcome_message_id;

-- name: ForgetGuildWelcome :exec
-- The language stays: the bot may come back.
UPDATE guilds
SET welcome_channel_id = NULL,
    welcome_message_id = NULL
WHERE guild_id = $1;
