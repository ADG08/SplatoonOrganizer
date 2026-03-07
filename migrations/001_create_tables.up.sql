-- +migrate Up

CREATE TABLE IF NOT EXISTS sondage_messages (
    week       TEXT PRIMARY KEY,
    message_id TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS dispos (
    user_id    TEXT    NOT NULL,
    day_index  SMALLINT NOT NULL, -- 0 = lundi ... 6 = dimanche
    slot_index SMALLINT NOT NULL, -- 0 = matin, 1 = après-midi, 2 = soir
    week       TEXT    NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT dispos_unique_per_week UNIQUE (user_id, day_index, slot_index, week)
);

