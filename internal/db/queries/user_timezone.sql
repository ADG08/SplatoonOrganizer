-- name: GetUserTimezone :one
SELECT timezone
FROM user_timezone
WHERE user_id = $1;

-- name: SetUserTimezone :exec
INSERT INTO user_timezone (user_id, timezone)
VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE
SET timezone   = EXCLUDED.timezone,
    updated_at = NOW();
