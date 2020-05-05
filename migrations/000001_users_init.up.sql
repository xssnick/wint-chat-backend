CREATE TABLE users (
   id bigserial primary key,
   phone bigint NOT NULL,
   name text default '',
   city text default '',
   country text default '',
   created_at timestamp NOT NULL
);