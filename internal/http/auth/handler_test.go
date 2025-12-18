package authhandler_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	authconfig "github.com/braginantonev/mhserver/internal/config/auth"
	authhandler "github.com/braginantonev/mhserver/internal/http/auth"
	"github.com/braginantonev/mhserver/internal/repository/database"
	"github.com/braginantonev/mhserver/internal/server"
	"github.com/braginantonev/mhserver/internal/service/auth"
	"github.com/go-sql-driver/mysql"
)

const TEST_JWT_SIG string = "test123"

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

func TestLogin(t *testing.T) {
	db, err := database.OpenDB(mysql.Config{
		User:                 "mhserver_tests",
		Passwd:               "",
		Net:                  "tcp",
		Addr:                 "127.0.0.1:3306",
		DBName:               "mhs_main_test",
		AllowNativePasswords: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name          string
		method        string
		user          TestUser
		expected_code int
		expected_body string
	}{
		{
			name:   "wrong method",
			method: http.MethodPost,
			user: TestUser{
				User:                auth.NewUser("not registered", "123"),
				IsConvertibleToJSON: true,
			},
			expected_code: http.StatusMethodNotAllowed,
			expected_body: "",
		},
		{
			name:   "normal login",
			method: http.MethodGet,
			user: TestUser{
				User:                auth.NewUser("login_handler_test1", "123"),
				Register:            true,
				IsConvertibleToJSON: true,
			},
			expected_code: http.StatusOK,
			expected_body: "",
		},
		{
			name:   "empty request",
			method: http.MethodGet,
			user: TestUser{
				User:                auth.NewUser("empty", "empty"),
				IsConvertibleToJSON: false, // Empty json request
			},
			expected_code: http.StatusBadRequest,
			expected_body: authhandler.ErrRequestBodyEmpty.Error(),
		},
		{
			name:   "bad password",
			method: http.MethodGet,
			user: TestUser{
				User:                auth.NewUser("login_handler_test1", "123456"), // Use user from 'normal login' test
				IsConvertibleToJSON: true,
			},
			expected_code: http.StatusBadRequest,
			expected_body: auth.ErrWrongPassword.Error(),
		},
		{
			name:   "user not found",
			method: http.MethodGet,
			user: TestUser{
				User:                auth.NewUser("not_registered", "123"),
				IsConvertibleToJSON: true,
			},
			expected_code: http.StatusBadRequest,
			expected_body: auth.ErrUserNotExist.Error(),
		},
	}

	handler := authhandler.NewAuthHandler(authconfig.AuthHandlerConfig{
		JWTSignature: TEST_JWT_SIG,
		DB:           db,
	})

	for _, test := range cases {
		if test.user.Register {
			err := auth.Register(test.user.User, db)
			if errors.Is(errors.Unwrap(err), auth.ErrInternal) {
				t.Fatal(err)
			}
		}

		t.Run(test.name, func(t *testing.T) {
			body, err := test.user.ToJSON()
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest(test.method, server.LOGIN_ENDPOINT, bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.Login(w, req)
			res := w.Result()
			defer func() { _ = res.Body.Close() }()

			if res.StatusCode != test.expected_code {
				t.Errorf("expected status code %d, but got %d", test.expected_code, res.StatusCode)
			}

			received_body, err := io.ReadAll(w.Body)
			if err != nil {
				t.Error(err)
			}

			if test.expected_code == http.StatusOK {
				if err := auth.CheckJWTUserMatch(test.user.Name, string(received_body), TEST_JWT_SIG); err != nil {
					t.Error(err)
				}
			} else {
				if string(received_body) != test.expected_body {
					t.Errorf("expected body: \"%s\"\nbut got: \"%s\"", test.expected_body, string(received_body))
				}
			}
		})
	}

	// Clear registered users
	for _, test := range cases {
		if _, err := db.Exec("delete from users where user=?", test.user.Name); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRegister(t *testing.T) {
	db, err := database.OpenDB(mysql.Config{
		User:                 "mhserver_tests",
		Passwd:               "",
		Net:                  "tcp",
		Addr:                 "127.0.0.1:3306",
		DBName:               "mhs_main_test",
		AllowNativePasswords: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name          string
		method        string
		user          TestUser
		expected_code int
		expected_body string
	}{
		{
			name:   "wrong method",
			method: http.MethodGet,
			user: TestUser{
				User:                auth.NewUser("not registered", "123"),
				IsConvertibleToJSON: true,
			},
			expected_code: http.StatusMethodNotAllowed,
			expected_body: "",
		},
		{
			name:   "normal register",
			method: http.MethodPost,
			user: TestUser{
				User:                auth.NewUser("register_handler_test1", "123"),
				IsConvertibleToJSON: true,
			},
			expected_code: http.StatusOK,
			expected_body: "",
		},
		{
			name:   "empty username",
			method: http.MethodPost,
			user: TestUser{
				User:                auth.NewUser("", "123"),
				IsConvertibleToJSON: true,
			},
			expected_code: http.StatusBadRequest,
			expected_body: auth.ErrNameIsEmpty.Error(),
		},
		{
			name:   "empty request",
			method: http.MethodPost,
			user: TestUser{
				User:                auth.NewUser("register_handler_test1", "123"),
				IsConvertibleToJSON: false,
			},
			expected_code: http.StatusBadRequest,
			expected_body: authhandler.ErrRequestBodyEmpty.Error(),
		},
	}

	handler := authhandler.NewAuthHandler(authconfig.AuthHandlerConfig{
		DB:           db,
		JWTSignature: TEST_JWT_SIG,
	})

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			req_body, err := test.user.ToJSON()
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest(test.method, server.REGISTER_ENDPOINT, bytes.NewReader(req_body))
			w := httptest.NewRecorder()

			handler.Register(w, req)
			res := w.Result()
			defer func() { _ = res.Body.Close() }()

			if res.StatusCode != test.expected_code {
				t.Errorf("expected status code %d, but got %d", test.expected_code, res.StatusCode)
			}

			resp_body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}

			if string(resp_body) != test.expected_body {
				t.Errorf("expected body: \"%s\"\nbut got \"%s\"", test.expected_body, string(resp_body))
			}

			row := db.QueryRow("select id from users where name=?", test.user.Name)
			if err := row.Scan(); err == sql.ErrNoRows {
				t.Error("user not found in db")
			}
		})
	}

	// Clear registered users
	for _, test := range cases {
		if _, err := db.Exec("delete from users where user=?", test.user.Name); err != nil {
			t.Fatal(err)
		}
	}
}
