-- +goose Up
CREATE TABLE users (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    auth_uid     text NOT NULL UNIQUE,
    display_name text NOT NULL DEFAULT '',
    timezone     text NOT NULL DEFAULT 'Asia/Tokyo',
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE medications (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name       text NOT NULL,
    dose_text  text NOT NULL DEFAULT '',
    timings    smallint[] NOT NULL,
    active     boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX medications_user_id_idx ON medications (user_id);

-- 服薬記録は追記専用。id はクライアント生成のUUID（再送時の冪等性のため）。
CREATE TABLE dose_logs (
    id             uuid PRIMARY KEY,
    user_id        uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    medication_id  uuid NOT NULL REFERENCES medications (id) ON DELETE CASCADE,
    timing         smallint NOT NULL,
    scheduled_date date NOT NULL,
    taken_at       timestamptz NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX dose_logs_user_date_idx ON dose_logs (user_id, scheduled_date);
CREATE INDEX dose_logs_slot_idx ON dose_logs (medication_id, scheduled_date, timing);

-- +goose Down
DROP TABLE dose_logs;
DROP TABLE medications;
DROP TABLE users;
