package eioh

import (
	"database/sql"
	"errors"
)

var (
	ErrTableDoesNotExist = errors.New("table does not exist")
	ErrNoPreviousVersion = errors.New("no previous version found")
)

type SqlBase interface {
	createVersionTableSql() string
	insertVersionSql() string
	dbVersionQuery(db *sql.DB) (*sql.Rows, error)
}

func baseByName(d string) SqlBase {
	switch d {
	case "mysql":
		return &MySqlBase{}
	}
	return nil
}

type MySqlBase struct{}

func (m MySqlBase) createVersionTableSql() string {
	return `CREATE TABLE db_version (
                ID serial NOT NULL,
                VERSION bigint NOT NULL,
                APPLIED boolean NOT NULL,
                CREATEDATE timestamp NULL default now(),
                PRIMARY KEY(id)
            );`
}

func (m MySqlBase) insertVersionSql() string {
	return "INSERT INTO db_version (VERSION, APPLIED) VALUES (?, ?);"
}

func (m MySqlBase) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query("SELECT VERSION,  APPLIED from db_version ORDER BY id DESC")

	if err != nil {
		return nil, ErrTableDoesNotExist
	}
	return rows, err
}
