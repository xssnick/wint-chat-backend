ALTER TABLE users
    ADD COLUMN reg_finished BOOLEAN default false,
    ADD COLUMN description TEXT default '',
    ADD COLUMN sex INTEGER default 0,
    ADD COLUMN birth timestamp default NULL;