//! Run this test with sudo

package auth_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	"github.com/braginantonev/mhserver/internal/application"
	"github.com/braginantonev/mhserver/pkg/auth"
	"github.com/braginantonev/mhserver/pkg/httperror"
	"golang.org/x/crypto/bcrypt"
)

func open_db() (*sql.DB, error) {
	app := application.NewApplication()
	DB, err := sql.Open("mysql", fmt.Sprintf("mhserver:%s@/%s", app.DB_Pass, app.ServerName))
	if err != nil {
		return nil, err
	}

	if err = DB.Ping(); err != nil {
		return nil, err
	}

	return DB, nil
}

func TestRegister(t *testing.T) {
	cases := []struct {
		name         string
		username     string
		password     string
		expected_err httperror.HttpError
		get_from_db  bool
	}{
		{
			name:         "Base register",
			username:     "register_test1",
			password:     "123",
			expected_err: httperror.NewEmptyHttpError(),
			get_from_db:  true,
		},
		{
			name:         "Empty name",
			username:     "",
			password:     "123",
			expected_err: httperror.NewExternalHttpError(auth.ErrNameIsEmpty, http.StatusBadRequest),
			get_from_db:  false,
		},
		{
			name:         "Already register",
			username:     "register_test2",
			password:     "123",
			expected_err: httperror.NewExternalHttpError(auth.ErrUserAlreadyExists, http.StatusContinue),
			get_from_db:  true,
		},
	}

	db, err := open_db()
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close()

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
	db, err := open_db()
	if err != nil {
		t.Error(err)
		return
	}

	jwt_signature := "test"

	cases := []struct {
		name          string
		user          auth.User
		expected_herr httperror.HttpError
		check_jwt     bool
	}{
		{
			name:          "Empty username",
			user:          auth.NewUser("", ""),
			expected_herr: httperror.NewExternalHttpError(auth.ErrNameIsEmpty, http.StatusBadRequest),
		},
		{
			name:          "Not registered",
			user:          auth.NewUser("unregistered user", "123"),
			expected_herr: httperror.NewExternalHttpError(auth.ErrUserNotExist, http.StatusNotFound),
		},
		{
			name:          "Wrong password",
			user:          auth.NewUser("login_test1", "123"),
			expected_herr: httperror.NewExternalHttpError(auth.ErrWrongPassword, http.StatusBadRequest),
		},
		{
			name:          "Normal login",
			user:          auth.NewUser("login_test2", "123"),
			expected_herr: httperror.NewEmptyHttpError(),
			check_jwt:     true,
		},
	}

	wrong_password_user := auth.NewUser("login_test1", "321")
	if herr := auth.Register(wrong_password_user, db); herr.Type != httperror.EMPTY {
		t.Error(herr.Error())
	}

	if herr := auth.Register(cases[3].user, db); herr.Type != httperror.EMPTY {
		t.Error(herr.Error())
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			token, herr := auth.Login(test.user, db, jwt_signature)
			if err := test.expected_herr.CompareWith(herr); err != nil {
				t.Error(err)
			}

			if !test.check_jwt {
				return
			}

			if err := auth.CheckJWTUserMatch(test.user.Name, token, jwt_signature); err != nil {
				t.Error(err)
			}
		})

		_, err := db.Exec("DELETE FROM users WHERE user = ?", test.user.Name)
		if err != nil {
			fmt.Println(err)
		}
	}

	_, err = db.Exec("DELETE FROM users WHERE user = ?", wrong_password_user.Name)
	if err != nil {
		fmt.Println(err)
	}
}
