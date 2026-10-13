-- name: GetWeeklyMessage :one
SELECT week, channel_id, message_id
FROM weekly_messages
WHERE guild_id = $1;

-- name: UpsertWeeklyMessage :exec
INSERT INTO weekly_messages (guild_id, week, channel_id, message_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (guild_id) DO UPDATE
SET week       = EXCLUDED.week,
    channel_id = EXCLUDED.channel_id,
    message_id = EXCLUDED.message_id;

-- name: GetUserAvailability :many
SELECT day_index, slot_index
FROM dispos
WHERE guild_id = $1
  AND week = $2
  AND user_id = $3
ORDER BY day_index, slot_index;

-- name: SetDayAvailability :exec
-- Replaces a member's slots of one day in a single round trip: the deleted rows (slots
-- not selected anymore) and the inserted ones (selected slots) never overlap. Selecting
-- at least one slot also clears the "unavailable for the week" status.
WITH removed AS (
    DELETE FROM dispos
    WHERE guild_id = @guild_id::text
      AND week = @week::text
      AND user_id = @user_id::text
      AND day_index = @day_index::smallint
      AND NOT (slot_index = ANY (@slots::smallint[]))
), available_again AS (
    DELETE FROM week_unavailable
    WHERE guild_id = @guild_id::text
      AND week = @week::text
      AND user_id = @user_id::text
      AND cardinality(@slots::smallint[]) > 0
)
INSERT INTO dispos (guild_id, week, user_id, day_index, slot_index)
SELECT @guild_id::text, @week::text, @user_id::text, @day_index::smallint, unnest(@slots::smallint[])
ON CONFLICT DO NOTHING;

-- name: GetAvailabilityCounts :many
SELECT day_index, slot_index, COUNT(*) AS count
FROM dispos
WHERE guild_id = $1
  AND week = $2
GROUP BY day_index, slot_index
ORDER BY day_index, slot_index;

-- name: GetAvailabilityUsers :many
SELECT day_index, slot_index, user_id
FROM dispos
WHERE guild_id = $1
  AND week = $2
ORDER BY day_index, slot_index, user_id;

-- name: DeleteAllUserAvailability :exec
DELETE FROM dispos
WHERE guild_id = $1
  AND week = $2
  AND user_id = $3;

-- name: DeleteWeekAvailability :exec
DELETE FROM dispos
WHERE guild_id = $1
  AND week = $2;

-- name: SetWeekUnavailable :exec
INSERT INTO week_unavailable (guild_id, week, user_id)
VALUES ($1, $2, $3)
ON CONFLICT (guild_id, week, user_id) DO NOTHING;

-- name: DeleteWeekUnavailable :exec
DELETE FROM week_unavailable
WHERE guild_id = $1
  AND week = $2
  AND user_id = $3;

-- name: IsUserWeekUnavailable :one
SELECT EXISTS (
    SELECT 1 FROM week_unavailable
    WHERE guild_id = $1 AND week = $2 AND user_id = $3
) AS unavailable;

-- name: GetWeekUnavailableUsers :many
SELECT user_id
FROM week_unavailable
WHERE guild_id = $1
  AND week = $2
ORDER BY user_id;
