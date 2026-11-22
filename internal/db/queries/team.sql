-- name: CreateTeam :one
INSERT INTO teams (guild_id, name, language)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetTeam :one
SELECT *
FROM teams
WHERE id = $1;

-- name: ListTeams :many
SELECT *
FROM teams
WHERE guild_id = $1
ORDER BY id;

-- name: ListTeamsWithChannel :many
SELECT *
FROM teams
WHERE channel_id IS NOT NULL AND active
ORDER BY id;

-- name: DeleteTeam :exec
DELETE FROM teams
WHERE id = $1;

-- name: RenameTeam :exec
UPDATE teams
SET name = $2
WHERE id = $1;

-- name: SetTeamChannel :exec
UPDATE teams
SET channel_id = $2,
    active     = TRUE
WHERE id = $1;

-- name: SetTeamRole :exec
UPDATE teams
SET role_id = $2
WHERE id = $1;

-- name: SetTeamMessageSettings :exec
UPDATE teams
SET slot_mode       = $2,
    slot_start_hour = $3,
    slot_end_hour   = $4,
    display_style   = $5,
    language        = $6
WHERE id = $1;

-- name: SetTeamSchedule :exec
UPDATE teams
SET publish_weekday = $2,
    publish_hour    = $3
WHERE id = $1;

-- name: SetTeamActive :exec
UPDATE teams
SET active = $2
WHERE id = $1;

-- name: SetGuildTeamsActive :many
-- Only writes the teams that change: on startup, every guild is announced.
UPDATE teams
SET active = $2
WHERE guild_id = $1 AND active <> $2
RETURNING id;

-- name: SetTeamTimezone :exec
UPDATE teams
SET timezone = $2
WHERE id = $1;

-- name: SetTeamMemberTimezones :exec
UPDATE teams
SET member_timezones = $2
WHERE id = $1;

-- name: SetTeamFullAt :exec
UPDATE teams
SET full_at = $2
WHERE id = $1;
