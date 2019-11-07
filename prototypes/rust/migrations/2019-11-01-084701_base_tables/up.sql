-- Your SQL goes here

create table if not exists hosts
(
    id       integer primary key,
    request  varchar not null,
    checksum varchar not null
)