package auth_handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/braginantonev/mhserver/internal/application"
	"github.com/braginantonev/mhserver/internal/server"
	auth_handlers "github.com/braginantonev/mhserver/internal/server/services/auth/handlers"
	"github.com/braginantonev/mhserver/pkg/auth"
	"github.com/braginantonev/mhserver/pkg/httperror"
)

type TestUser struct {
	auth.User
	skipBodyCheck bool
}

func (user TestUser) ToJSON() ([]byte, error) {
	if user.skipBodyCheck {
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
		name                  string
		method                string
		user                  TestUser
		expected_code         int
		expected_body_message []byte
	}{
		{
			name:   "normal login",
			method: http.MethodGet,
			user: TestUser{
				User:          auth.NewUser("login_handler_test1", "123"),
				skipBodyCheck: false,
			},
			expected_code:         http.StatusOK,
			expected_body_message: []byte(""),
		},
	}

	db, err := open_db()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()

	handler := auth_handlers.NewAuthHandler(auth_handlers.Config{
		JWTSignature: "test123",
		DB:           db,
	})

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			reg_err := auth.Register(test.user.User, db)
			if reg_err.Type != httperror.EMPTY {
				t.Error(reg_err)
				return
			}

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

			//
			if test.user.skipBodyCheck {
				return
			}
		})
	}
}
