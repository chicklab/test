
-- +eioh up
-- SQL in section 'up' is executed when this migration is applied

CREATE TABLE db_version4 (
                ID serial NOT NULL,
                VERSION bigint NOT NULL,
                APPLIED boolean NOT NULL,
                CREATEDATE timestamp NULL default now(),
                PRIMARY KEY(id)
            );

-- +eioh down
-- SQL section 'down' is executed when this migration is rolled back

DROP TABLE db_version4;