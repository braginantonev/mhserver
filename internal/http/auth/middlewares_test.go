package authhandler_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/braginantonev/mhserver/internal/application"
	authconfig "github.com/braginantonev/mhserver/internal/config/auth"
	authmiddleware "github.com/braginantonev/mhserver/internal/http/auth"
	"github.com/braginantonev/mhserver/internal/repository/database"
	"github.com/braginantonev/mhserver/internal/services/auth"
	"github.com/braginantonev/mhserver/pkg/httpcontextkeys"
	"github.com/golang-jwt/jwt/v5"
)

type Token struct {
	string

	// If not wrong, use auth.Login()
	IsWrong bool
}

/*
args pos:

	0 - jwt signature (string)
	1 - jwt_signing_method (jwt.SigningMethod)
	2 - nbf (int64)
	3 - exp (int64)
	4 - iat (int64)
*/
func NewWrongToken(bad_username string, args ...any) Token {
	var token_str string

	if bad_username != "" {
		token := jwt.NewWithClaims(args[1].(jwt.SigningMethod), jwt.MapClaims{
			"name": bad_username,
			"nbf":  args[2].(int64),
			"exp":  args[3].(int64),
			"iat":  args[4].(int64),
		})

		var err error
		token_str, err = token.SignedString([]byte(args[0].(string)))
		if err != nil {
			fmt.Println(err)
		}
	}

	return Token{
		string:  token_str,
		IsWrong: true,
	}
}

func TestWithAuth(t *testing.T) {
	app := application.NewApplication()
	db, err := database.OpenDB("mhserver", app.DB_Pass, app.ServerName)
	if err != nil {
		t.Fatal(err)
	}

	time_now := time.Now()

	cases := [...]struct {
		name          string
		user          TestUser
		token         Token
		expected_code int
		expected_body string
	}{
		{
			name:  "normal auth",
			token: Token{},
			user: TestUser{
				User:     auth.NewUser("with_auth_middleware_test1", "123"),
				Register: true,
			},
			expected_code: http.StatusOK, // Not set
			expected_body: "",            // Not set
		},
		{
			name:  "empty authorization token",
			token: NewWrongToken(""),
			user: TestUser{
				User: auth.NewUser("123", "123"),
			},
			expected_code: http.StatusUnauthorized,
			expected_body: authmiddleware.ErrUserNotAuthorized.Error(),
		},
		{
			name:  "wrong token signature",
			token: NewWrongToken("123", "wrong signature", jwt.SigningMethodHS256, time_now.Unix(), time_now.Add(time.Minute).Unix(), time_now.Unix()),
			user: TestUser{
				User: auth.NewUser("123", "123"),
			},
			expected_code: http.StatusBadRequest,
			expected_body: authmiddleware.ErrJwtSignatureInvalid.Error(),
		},
		{
			name:  "expired token",
			token: NewWrongToken("123", app.JWTSignature, jwt.SigningMethodHS256, time_now.Unix()-60, time_now.Unix()-30, time_now.Unix()-60),
			user: TestUser{
				User: auth.NewUser("123", "123"),
			},
			expected_code: http.StatusUnauthorized,
			expected_body: authmiddleware.ErrAuthorizationExpired.Error(),
		},
	}

	middleware := authmiddleware.NewAuthMiddleware(authconfig.AuthMiddlewareConfig{
		JWTSignature: app.JWTSignature,
	})

	for _, test := range cases {
		if test.user.Register {
			err := auth.Register(test.user.User, db)
			if errors.Is(errors.Unwrap(err), auth.ErrInternal) {
				t.Fatal(err)
			}
		}

		t.Run(test.name, func(t *testing.T) {
			if !test.token.IsWrong {
				var err error
				test.token.string, err = auth.Login(test.user.User, db, app.JWTSignature)
				if errors.Is(errors.Unwrap(err), auth.ErrInternal) {
					t.Fatal(err)
				}
			}

			check_fn := func(w http.ResponseWriter, r *http.Request) {
				res, ok := r.Context().Value(httpcontextkeys.USERNAME).(string)
				if ok {
					if res != test.user.Name {
						t.Errorf("expected name: \"%s\", but got \"%s\"", test.user.Name, res)
					}
				} else {
					t.Errorf("context value is not string. value=%v", res)
				}
			}

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Add("Authorization", test.token.string)

			w := httptest.NewRecorder()

			middleware.WithAuth(check_fn).ServeHTTP(w, req)

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
				t.Errorf("expected body \"%s\"\nbut got \"%s\"", test.expected_body, string(resp_body))
			}
		})
	}
}
