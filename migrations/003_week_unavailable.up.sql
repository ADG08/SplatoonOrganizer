-- +migrate Up

CREATE TABLE IF NOT EXISTS week_unavailable (
    user_id    TEXT    NOT NULL,
    week       TEXT    NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT week_unavailable_pk PRIMARY KEY (user_id, week)
);
