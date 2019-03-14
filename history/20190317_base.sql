-- +goose Up

CREATE TABLE db_version2 (
                ID serial NOT NULL,
                VERSION bigint NOT NULL,
                APPLIED boolean NOT NULL,
                CREATEDATE timestamp NULL default now(),
                PRIMARY KEY(id)
            );

-- +goose Down

DROP TABLE db_version2;