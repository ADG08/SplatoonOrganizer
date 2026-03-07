-- name: GetGuildConfig :one
SELECT guild_id, channel_id, role_id
FROM guild_config
WHERE guild_id = $1;

-- name: SetGuildConfigChannel :exec
INSERT INTO guild_config (guild_id, channel_id)
VALUES ($1, $2)
ON CONFLICT (guild_id) DO UPDATE
SET channel_id = EXCLUDED.channel_id;

-- name: SetGuildConfigRole :exec
INSERT INTO guild_config (guild_id, role_id)
VALUES ($1, $2)
ON CONFLICT (guild_id) DO UPDATE
SET role_id = EXCLUDED.role_id;
