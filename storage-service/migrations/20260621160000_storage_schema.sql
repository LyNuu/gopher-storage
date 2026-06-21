-- +goose Up
drop table if exists user_storage;

create type storage_type as enum ('personal', 'global');
create type access_level as enum ('read', 'write');

create table storages
(
    id                  uuid primary key      default gen_random_uuid(),
    owner_id            uuid         not null,
    name                varchar(255) not null,
    type                storage_type not null,
    max_size_bytes      bigint       not null,
    used_bytes          bigint       not null default 0,
    max_file_size_bytes bigint       not null default 0,
    allowed_mime_types  text[]       not null default '{}',
    created_at          timestamptz  not null default now(),
    updated_at          timestamptz  not null default now()
);

create index idx_storages_owner_id on storages (owner_id);

create table storage_access
(
    storage_id uuid         not null references storages (id) on delete cascade,
    user_id    uuid         not null,
    level      access_level not null,
    granted_by uuid         not null,
    created_at timestamptz  not null default now(),
    primary key (storage_id, user_id)
);

create table files
(
    id           uuid primary key      default gen_random_uuid(),
    storage_id   uuid         not null references storages (id) on delete cascade,
    name         varchar(512) not null,
    size_bytes   bigint       not null,
    content_type varchar(255) not null,
    uploaded_by  uuid         not null,
    created_at   timestamptz  not null default now(),
    updated_at   timestamptz  not null default now(),
    unique (storage_id, name)
);

create index idx_files_storage_id on files (storage_id);

create table file_shares
(
    token      varchar(64) primary key,
    file_id    uuid        not null references files (id) on delete cascade,
    created_by uuid        not null,
    expires_at timestamptz not null,
    created_at timestamptz not null default now()
);

create index idx_file_shares_file_id on file_shares (file_id);

-- +goose Down
drop table if exists file_shares;
drop table if exists files;
drop table if exists storage_access;
drop table if exists storages;
drop type if exists access_level;
drop type if exists storage_type;
