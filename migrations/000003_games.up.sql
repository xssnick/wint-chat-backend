CREATE TABLE games (
                       id bigserial primary key,
                       owner bigint REFERENCES users (id),
                       winner bigint REFERENCES users (id),
                       created_at timestamp NOT NULL,
                       finished_at timestamp default NULL
);

CREATE TABLE plays (
                       id bigserial primary key,
                       player bigint REFERENCES users (id),
                       game bigint REFERENCES games (id),
                       lost_at timestamp default NULL
);

CREATE TABLE messages (
                       id bigserial primary key,
                       owner bigint REFERENCES users (id),
                       target bigint REFERENCES users (id),
                       game bigint REFERENCES games (id),
                       msg text NOT NULL,
                       created_at timestamp NOT NULL
);

CREATE TABLE likes (
                       owner bigint REFERENCES users (id),
                       message bigint REFERENCES messages (id)
);