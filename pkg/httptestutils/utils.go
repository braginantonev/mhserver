package httptestutils

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/braginantonev/mhserver/pkg/auth"
)

type TestUser struct {
	auth.User
	Register            bool
	IsConvertibleToJSON bool
}

func (user TestUser) ToJSON() ([]byte, error) {
	if !user.IsConvertibleToJSON {
		return []byte(""), nil
	}

	return json.Marshal(user)
}

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
