-- +goose Up
create type user_role as enum ('user', 'admin');

create table users
(
    id            uuid primary key      default gen_random_uuid(),
    email         varchar(255) not null unique,
    password_hash varchar(255) not null,
    role          user_role    not null default 'user',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users (email);
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS user_role;
-- +goose StatementEnd
