-- +eioh Up

CREATE TABLE db_version3 (
                ID serial NOT NULL,
                VERSION bigint NOT NULL,
                APPLIED boolean NOT NULL,
                CREATEDATE timestamp NULL default now(),
                PRIMARY KEY(id)
            );

-- +eioh Down

DROP TABLE db_version3;