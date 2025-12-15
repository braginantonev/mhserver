//! Run this test with sudo

package auth_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/braginantonev/mhserver/internal/application"
	"github.com/braginantonev/mhserver/internal/services/auth"
	"github.com/braginantonev/mhserver/pkg/httptestutils"
	"golang.org/x/crypto/bcrypt"
)

func TestRegister(t *testing.T) {
	cases := []struct {
		name         string
		username     string
		password     string
		expected_err error
		get_from_db  bool
	}{
		{
			name:         "Base register",
			username:     "register_test1",
			password:     "123",
			expected_err: nil,
			get_from_db:  true,
		},
		{
			name:         "Empty name",
			username:     "",
			password:     "123",
			expected_err: auth.ErrNameIsEmpty,
			get_from_db:  false,
		},
		{
			name:         "Already register",
			username:     "register_test2",
			password:     "123",
			expected_err: auth.ErrUserAlreadyExists,
			get_from_db:  true,
		},
	}

	app := application.NewApplication()
	db, err := httptestutils.OpenDB("mhserver", app.DB_Pass, app.ServerName)
	if err != nil {
		t.Fatal(err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(auth.INSERT_USER, "register_test2", string(hash))
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			err := auth.Register(auth.User{
				Name:     test.username,
				Password: test.password,
			}, db)

			if !errors.Is(err, test.expected_err) {
				t.Errorf("expected error: %s, but got: %s", test.expected_err, err)
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
	jwt_signature := "test"

	cases := []struct {
		name         string
		user         auth.User
		expected_err error
		check_jwt    bool
	}{
		{
			name:         "Empty username",
			user:         auth.NewUser("", ""),
			expected_err: auth.ErrNameIsEmpty,
		},
		{
			name:         "Not registered",
			user:         auth.NewUser("unregistered user", "123"),
			expected_err: auth.ErrUserNotExist,
		},
		{
			name:         "Wrong password",
			user:         auth.NewUser("login_test1", "123"),
			expected_err: auth.ErrWrongPassword,
		},
		{
			name:         "Normal login",
			user:         auth.NewUser("login_test2", "123"),
			expected_err: nil,
			check_jwt:    true,
		},
	}

	app := application.NewApplication()
	db, err := httptestutils.OpenDB("mhserver", app.DB_Pass, app.ServerName)
	if err != nil {
		t.Fatal(err)
	}

	wrong_password_user := auth.NewUser("login_test1", "321")
	if err := auth.Register(wrong_password_user, db); err != nil {
		t.Fatal(err)
	}

	if err := auth.Register(cases[3].user, db); err != nil {
		t.Fatal(err)
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			token, err := auth.Login(test.user, db, jwt_signature)
			if !errors.Is(err, test.expected_err) {
				t.Errorf("expected error: %s, but got: %s", test.expected_err, err)
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
