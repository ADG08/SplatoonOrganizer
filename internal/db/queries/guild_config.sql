-- name: GetGuildConfig :one
SELECT guild_id, channel_id, role_id, slot_mode, slot_start_hour, slot_end_hour, display_style
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

-- name: SetGuildConfigSlotSettings :exec
INSERT INTO guild_config (guild_id, slot_mode, slot_start_hour, slot_end_hour, display_style)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (guild_id) DO UPDATE
SET slot_mode       = EXCLUDED.slot_mode,
    slot_start_hour = EXCLUDED.slot_start_hour,
    slot_end_hour   = EXCLUDED.slot_end_hour,
    display_style   = EXCLUDED.display_style;

-- name: ListGuildConfigsWithChannel :many
SELECT guild_id, channel_id, role_id, slot_mode, slot_start_hour, slot_end_hour, display_style
FROM guild_config
WHERE channel_id IS NOT NULL
ORDER BY guild_id;
