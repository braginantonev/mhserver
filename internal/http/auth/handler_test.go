package authhttp_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/braginantonev/mhserver/internal/config"
	authconfig "github.com/braginantonev/mhserver/internal/config/auth"
	authhttp "github.com/braginantonev/mhserver/internal/http/auth"
	"github.com/braginantonev/mhserver/internal/repository/database"
	"github.com/braginantonev/mhserver/internal/server"
	"github.com/braginantonev/mhserver/internal/service/auth"
	"github.com/go-sql-driver/mysql"
)

const (
	TEST_JWT_SIG               string = "test123"
	TEST_REGISTER_SECRET_KEY   string = "TEST_SECRET_KEY"
	INSERT_REGISTER_SECRET_KEY string = "INSERT INTO register_secret_keys (secret_key) VALUES (?)"
)

func InsertRegisterKeyToDB(db *sql.DB, secret_key string) error {
	_, err := db.Exec(INSERT_REGISTER_SECRET_KEY, secret_key)
	return err
}

type TestUser struct {
	auth.User
	RegisterSecretKey   string `json:"key"`
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
			name:   "normal login",
			method: http.MethodGet,
			user: TestUser{
				User:                auth.NewUser("login_handler_test1", "123"),
				RegisterSecretKey:   TEST_REGISTER_SECRET_KEY,
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
			expected_body: authhttp.ErrRequestBodyEmpty.Description(),
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

	handler := authhttp.NewHandler(authconfig.AuthHandlerConfig{
		JWTSignature: TEST_JWT_SIG,
		Requests: config.RequestsConfig{
			MaxInInterval:   5,
			LimiterInterval: time.Second,
		},
		DB: db,
	})

	for _, test := range cases {
		if test.user.Register {
			if err := InsertRegisterKeyToDB(db, TEST_REGISTER_SECRET_KEY); err != nil {
				t.Fatalf("failed to insert register key to DB: %v", err)
			}

			err := auth.Register(auth.NewRegisterUser(test.user.User, test.user.RegisterSecretKey), db)
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
		user          TestUser
		expected_code int
		expected_body string
	}{
		{
			name: "normal register",
			user: TestUser{
				User:                auth.NewUser("register_handler_test1", "123"),
				RegisterSecretKey:   TEST_REGISTER_SECRET_KEY,
				IsConvertibleToJSON: true,
			},
			expected_code: http.StatusOK,
			expected_body: "",
		},
		{
			name: "empty username",
			user: TestUser{
				User:                auth.NewUser("", "123"),
				IsConvertibleToJSON: true,
			},
			expected_code: http.StatusBadRequest,
			expected_body: authhttp.ErrUsernameEmpty.Description(),
		},
		{
			name: "empty secret key",
			user: TestUser{
				User:                auth.NewUser("123", "123"),
				IsConvertibleToJSON: true,
			},
			expected_code: http.StatusBadRequest,
			expected_body: authhttp.ErrRegSecretKeyEmpty.Description(),
		},
		{
			name: "empty request",
			user: TestUser{
				User:                auth.NewUser("register_handler_test1", "123"),
				IsConvertibleToJSON: false,
			},
			expected_code: http.StatusBadRequest,
			expected_body: authhttp.ErrRequestBodyEmpty.Description(),
		},
	}

	handler := authhttp.NewHandler(authconfig.AuthHandlerConfig{
		DB: db,
		Requests: config.RequestsConfig{
			MaxInInterval:   5,
			LimiterInterval: time.Second,
		},
		JWTSignature: TEST_JWT_SIG,
	})

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if test.user.RegisterSecretKey != "" {
				if err := InsertRegisterKeyToDB(db, TEST_REGISTER_SECRET_KEY); err != nil {
					t.Fatalf("failed to insert register key to DB: %v", err)
				}
			}

			req_body, err := test.user.ToJSON()
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest(http.MethodPost, server.REGISTER_ENDPOINT, bytes.NewReader(req_body))
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
