package database

import (
	"database/sql"

	"github.com/go-sql-driver/mysql"
)

func OpenDB(cfg mysql.Config) (*sql.DB, error) {
	DB, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}

	if err = DB.Ping(); err != nil {
		return nil, err
	}

	return DB, nil
}
