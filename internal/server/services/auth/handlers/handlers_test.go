package auth_handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/braginantonev/mhserver/internal/application"
	"github.com/braginantonev/mhserver/internal/server"
	"github.com/braginantonev/mhserver/internal/server/services"
	auth_handlers "github.com/braginantonev/mhserver/internal/server/services/auth/handlers"
	"github.com/braginantonev/mhserver/pkg/auth"
	"github.com/braginantonev/mhserver/pkg/httperror"
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

func TestLogin(t *testing.T) {
	cases := []struct {
		name          string
		method        string
		user          TestUser
		expected_code int
		expected_body string
	}{
		{
			name:   "bad method",
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
			expected_body: services.MESSAGE_REQUEST_BODY_EMPTY,
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

	db, err := open_db()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()

	handler := auth_handlers.NewAuthHandler(auth_handlers.Config{
		JWTSignature: TEST_JWT_SIG,
		DB:           db,
	})

	for _, test := range cases {
		if test.user.Register {
			reg_err := auth.Register(test.user.User, db)
			if reg_err.Type != httperror.EMPTY && reg_err.StatusCode != http.StatusContinue {
				t.Error(reg_err)
				return
			}
		}

		t.Run(test.name, func(t *testing.T) {
			body, err := test.user.ToJSON()
			if err != nil {
				t.Error(err)
				return
			}

			req := httptest.NewRequest(test.method, server.LOGIN_ENDPOINT, bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.Login(w, req)
			res := w.Result()

			if res.StatusCode != test.expected_code {
				t.Errorf("expected status code %d, but got %d", test.expected_code, res.StatusCode)
			}

			goted_body, err := io.ReadAll(w.Body)
			if err != nil {
				t.Error(err)
			}

			if test.expected_code == http.StatusOK {
				if err := auth.CheckJWTUserMatch(test.user.Name, string(goted_body), TEST_JWT_SIG); err != nil {
					t.Error(err)
				}
			} else {
				if string(goted_body) != test.expected_body {
					t.Errorf("expected body: \"%s\"\nbut got: \"%s\"", test.expected_body, string(goted_body))
				}
			}
		})
	}

	for _, test := range cases {
		if _, err := db.Exec("delete from users where user=?", test.user.Name); err != nil {
			t.Fatal(err)
		}
	}
}
