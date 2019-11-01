create table if not exists hosts
(
    id       integer primary key,
    request  varchar not null,
    checksum varchar not null
);

INSERT INTO hosts VALUES (1, '{"req":"pkg1"}', 'abcd');
INSERT INTO hosts VALUES (2, '{"req":"pkg2"}', 'efgh');
