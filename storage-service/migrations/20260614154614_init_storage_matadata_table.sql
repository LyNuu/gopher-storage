-- +goose Up
-- +goose StatementBegin
create table if not exists user_storage
(
    id             uuid primary key not null,
    max_size_bytes BIGINT           not null,
    used_bytes     BIGINT           not null default 0,
    created_at     TIMESTAMP        NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP        NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists user_storage
-- +goose StatementEnd
