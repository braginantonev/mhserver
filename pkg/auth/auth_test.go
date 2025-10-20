//! *************************
//!
//! Run this test with sudo!
//!
//! *************************

package auth_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	"github.com/braginantonev/mhserver/internal/application"
	"github.com/braginantonev/mhserver/pkg/auth"
	types "github.com/braginantonev/mhserver/pkg/handler_types"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func open_db(pass string, server_name string) (*sql.DB, error) {
	DB, err := sql.Open("mysql", fmt.Sprintf("mhserver:%s@/%s", pass, server_name))
	if err != nil {
		return nil, err
	}

	if err = DB.Ping(); err != nil {
		return nil, err
	}

	return DB, nil
}

// Todo: Добавить тесты для Login
func TestRegister(t *testing.T) {
	err := godotenv.Load("../../.env")
	if err != nil {
		fmt.Println(err)
	}

	app := application.NewApplication()

	cases := []struct {
		name         string
		username     string
		password     string
		expected_err types.HandlerError
		get_from_db  bool
	}{
		{
			name:         "Base register",
			username:     "register_test1",
			password:     "123",
			expected_err: types.NewEmptyHandlerError(),
			get_from_db:  true,
		},
		{
			name:         "Empty name",
			username:     "",
			password:     "123",
			expected_err: types.NewExternalHandlerError(auth.ErrNameIsEmpty, http.StatusBadRequest),
			get_from_db:  false,
		},
		{
			name:         "Already register",
			username:     "register_test2",
			password:     "123",
			expected_err: types.NewExternalHandlerError(auth.ErrUserAlreadyExists, http.StatusContinue),
			get_from_db:  true,
		},
	}

	db, err := open_db(app.DB_Pass, app.ServerName)
	if err != nil {
		t.Error(err)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("123"), bcrypt.DefaultCost)
	if err != nil {
		t.Error(err)
		return
	}
	_, err = db.Exec(auth.INSERT_USER, "register_test2", string(hash))
	if err != nil {
		t.Error(err)
		return
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			herr := auth.Register(auth.User{
				Name:     test.username,
				Password: test.password,
			}, db)

			if err := test.expected_err.CompareWith(herr); err != nil {
				t.Error(err)
			}

			if !test.get_from_db {
				return
			}

			db_user := auth.User{}
			row := db.QueryRow(auth.SELECT_USER, test.username)

			if err = row.Scan(&db_user.Name, &db_user.Password); err != nil {
				t.Error(err)
			}

			if db_user.Name != test.username {
				t.Errorf("expected name %s, but got %s", test.username, db_user.Name)
			}

			if err = bcrypt.CompareHashAndPassword([]byte(db_user.Password), []byte(test.password)); err != nil {
				t.Log(db_user.Password)
				t.Errorf("password incorrect. error=%s", err.Error())
			}
		})

		_, err := db.Exec("DELETE FROM users WHERE user = ?", test.username)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func TestLogin(t *testing.T) {

}
