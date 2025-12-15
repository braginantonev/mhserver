package database

import (
	"database/sql"
	"fmt"
)

func OpenDB(user string, user_pass string, database string) (*sql.DB, error) {
	DB, err := sql.Open("mysql", fmt.Sprintf("%s:%s@/%s", user, user_pass, database))
	if err != nil {
		return nil, err
	}

	if err = DB.Ping(); err != nil {
		return nil, err
	}

	return DB, nil
}
