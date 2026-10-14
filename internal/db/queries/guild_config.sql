-- name: GetGuildConfig :one
SELECT guild_id, channel_id, role_id, slot_mode, slot_start_hour, slot_end_hour, display_style, language,
       publish_weekday, publish_hour, active, timezone
FROM guild_config
WHERE guild_id = $1;

-- name: SetGuildConfigChannel :exec
INSERT INTO guild_config (guild_id, channel_id)
VALUES ($1, $2)
ON CONFLICT (guild_id) DO UPDATE
SET channel_id = EXCLUDED.channel_id,
    active     = TRUE;

-- name: SetGuildConfigRole :exec
INSERT INTO guild_config (guild_id, role_id)
VALUES ($1, $2)
ON CONFLICT (guild_id) DO UPDATE
SET role_id = EXCLUDED.role_id;

-- name: SetGuildConfigMessageSettings :exec
INSERT INTO guild_config (guild_id, slot_mode, slot_start_hour, slot_end_hour, display_style, language)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (guild_id) DO UPDATE
SET slot_mode       = EXCLUDED.slot_mode,
    slot_start_hour = EXCLUDED.slot_start_hour,
    slot_end_hour   = EXCLUDED.slot_end_hour,
    display_style   = EXCLUDED.display_style,
    language        = EXCLUDED.language;

-- name: ListGuildConfigsWithChannel :many
SELECT guild_id, channel_id, role_id, slot_mode, slot_start_hour, slot_end_hour, display_style, language,
       publish_weekday, publish_hour, active, timezone
FROM guild_config
WHERE channel_id IS NOT NULL AND active
ORDER BY guild_id;

-- name: SetGuildConfigSchedule :exec
INSERT INTO guild_config (guild_id, publish_weekday, publish_hour)
VALUES ($1, $2, $3)
ON CONFLICT (guild_id) DO UPDATE
SET publish_weekday = EXCLUDED.publish_weekday,
    publish_hour    = EXCLUDED.publish_hour;

-- name: SetGuildConfigActive :exec
UPDATE guild_config
SET active = $2
WHERE guild_id = $1;

-- name: SetGuildConfigTimezone :exec
INSERT INTO guild_config (guild_id, timezone)
VALUES ($1, $2)
ON CONFLICT (guild_id) DO UPDATE
SET timezone = EXCLUDED.timezone;
