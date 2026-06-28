-- name: UpsertSondageMessage :exec
INSERT INTO sondage_messages (week, message_id)
VALUES ($1, $2)
ON CONFLICT (week) DO UPDATE
SET message_id = EXCLUDED.message_id;

-- name: GetSondageMessageByWeek :one
SELECT week, message_id
FROM sondage_messages
WHERE week = $1;

-- name: ListSondageMessageIDs :many
SELECT message_id FROM sondage_messages;

-- name: GetUserAvailability :many
SELECT day_index, slot_index
FROM dispos
WHERE week = $1
  AND user_id = $2
ORDER BY day_index, slot_index;

-- name: InsertAvailability :exec
INSERT INTO dispos (user_id, day_index, slot_index, week)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, day_index, slot_index, week) DO NOTHING;

-- name: DeleteAvailability :exec
DELETE FROM dispos
WHERE user_id = $1
  AND day_index = $2
  AND slot_index = $3
  AND week = $4;

-- name: GetAvailabilityCounts :many
SELECT day_index, slot_index, COUNT(*) AS count
FROM dispos
WHERE week = $1
GROUP BY day_index, slot_index
ORDER BY day_index, slot_index;

-- name: GetAvailabilityUsers :many
SELECT day_index, slot_index, user_id
FROM dispos
WHERE week = $1
ORDER BY day_index, slot_index, user_id;

-- name: DeleteAllUserAvailability :exec
DELETE FROM dispos
WHERE user_id = $1
  AND week = $2;

-- name: SetWeekUnavailable :exec
INSERT INTO week_unavailable (user_id, week)
VALUES ($1, $2)
ON CONFLICT (user_id, week) DO NOTHING;

-- name: DeleteWeekUnavailable :exec
DELETE FROM week_unavailable
WHERE user_id = $1
  AND week = $2;

-- name: IsUserWeekUnavailable :one
SELECT EXISTS (
    SELECT 1 FROM week_unavailable
    WHERE user_id = $1 AND week = $2
) AS unavailable;

-- name: GetWeekUnavailableUsers :many
SELECT user_id
FROM week_unavailable
WHERE week = $1
ORDER BY user_id;

