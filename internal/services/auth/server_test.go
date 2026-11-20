package auth_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/braginantonev/mhserver/internal/services/auth"
	pb "github.com/braginantonev/mhserver/proto/gen/auth"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const (
	TEST_REGISTER_SECRET_KEY   string = "TEST_SECRET_KEY"
	INSERT_REGISTER_SECRET_KEY string = "INSERT INTO register_secret_keys (secret_key) VALUES (?)"

	JWT_SIGNATURE  string = "123zxc"
	WORKSPACE_PATH string = "/tmp/mhserver_tests"
)

type bcryptHash struct{}

func (hash bcryptHash) Match(v driver.Value) bool {
	pass, ok := v.(string)
	if !ok {
		return false
	}

	// bcrypt hash has `$2` prefix and mostly length = 60
	return strings.HasPrefix(pass, "$2") && len(pass) == 60
}

func checkJWTUserMatch(username, token, signature string) error {
	parsed, err := auth.ParseStringJWT(token, signature)
	if err != nil {
		return err
	}

	if claims, ok := parsed.Claims.(jwt.MapClaims); ok {
		if claims["name"] != username {
			return fmt.Errorf("expected user name: `%s`, but got `%s`", username, claims["name"])
		}
	} else {
		return errors.New("failed get jwt claims")
	}
	return nil
}

func createClient(ctx context.Context, config auth.AuthServiceConfig, db *sql.DB) (pb.AuthServiceClient, error) {
	grpc_server := grpc.NewServer()
	pb.RegisterAuthServiceServer(grpc_server, auth.NewAuthServer(config, db))

	lis := bufconn.Listen(1024 * 1024)
	go func() { _ = grpc_server.Serve(lis) }()

	grpc_connection, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return pb.NewAuthServiceClient(grpc_connection), nil
}

func TestRegister(t *testing.T) {
	config := auth.NewAuthServiceConfig(JWT_SIGNATURE)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed create mock of db: %v", err)
	}

	client, err := createClient(t.Context(), config, db)
	if err != nil {
		t.Fatalf("failed create client: %v", err)
	}

	cases := [...]struct {
		name         string
		setup_mock   func(mock sqlmock.Sqlmock)
		reg_info     *pb.RegisterRequest
		expected_err error
	}{
		{
			name:       "long username",
			setup_mock: func(mock sqlmock.Sqlmock) {},
			reg_info: &pb.RegisterRequest{
				User: &pb.User{
					Name:     "[Cop Killers] X1_BestCockSucker_1X",
					Password: "123",
				},
				SecretKey: TEST_REGISTER_SECRET_KEY,
			},
			expected_err: auth.ErrNameTooLong,
		},
		{
			name: "already registered",
			setup_mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id"}).AddRow(0)
				mock.ExpectQuery("^SELECT (.+) FROM users WHERE user = (.+)$").WithArgs("Arthur Morgan").WillReturnRows(rows)
			},
			reg_info: &pb.RegisterRequest{
				User: &pb.User{
					Name:     "Arthur Morgan",
					Password: "123",
				},
				SecretKey: TEST_REGISTER_SECRET_KEY,
			},
			expected_err: auth.ErrUserAlreadyExists,
		},
		{
			name: "key not found",
			setup_mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT (.+) FROM users WHERE user = (.+)$").WithArgs(sqlmock.AnyArg()).WillReturnError(sql.ErrNoRows)
				mock.ExpectQuery("^SELECT (.+) FROM register_secret_keys WHERE secret_key = (.+)$").WithArgs(sqlmock.AnyArg()).WillReturnError(sql.ErrNoRows)
			},
			reg_info: &pb.RegisterRequest{
				User: &pb.User{
					Name:     "Dracula",
					Password: "123",
				},
				SecretKey: "some undefined register key",
			},
			expected_err: auth.ErrRegSecretKeyNotFound,
		},
		{
			name: "base register",
			setup_mock: func(mock sqlmock.Sqlmock) {
				id := 0
				rows := sqlmock.NewRows([]string{"id"}).AddRow(id)
				mock.ExpectQuery("^SELECT (.+) FROM users WHERE user = (.+)$").WithArgs(sqlmock.AnyArg()).WillReturnError(sql.ErrNoRows)
				mock.ExpectQuery("^SELECT (.+) FROM register_secret_keys WHERE secret_key = (.+)$").WithArgs(TEST_REGISTER_SECRET_KEY).WillReturnRows(rows)

				mock.ExpectBegin()
				mock.ExpectExec("^INSERT INTO users ((.+), (.+)) VALUES ((.+), (.+))").WithArgs("V", bcryptHash{}).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("^DELETE FROM register_secret_keys WHERE id = (.+)$").WithArgs(id).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
			reg_info: &pb.RegisterRequest{
				User: &pb.User{
					Name:     "V",
					Password: "123",
				},
				SecretKey: TEST_REGISTER_SECRET_KEY,
			},
			expected_err: nil,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			test.setup_mock(mock)

			_, err := client.Register(t.Context(), test.reg_info)
			if !errors.Is(err, test.expected_err) {
				t.Errorf("expected error: %v, but got: %v", test.expected_err, err)
			}
		})
	}

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unfulfilled expectations: %v", err)
	}
}

func TestLogin(t *testing.T) {
	config := auth.NewAuthServiceConfig(JWT_SIGNATURE)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed create mock of db: %v", err)
	}

	client, err := createClient(t.Context(), config, db)
	if err != nil {
		t.Fatalf("failed create client: %v", err)
	}

	const GOOD_PASSWORD string = "123 (of course not)"

	cases := [...]struct {
		name         string
		setup_mock   func(mock sqlmock.Sqlmock)
		user         *pb.User
		expected_err error
	}{
		{
			name: "not registered",
			setup_mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT password FROM users WHERE user = (.+)$").WithArgs(sqlmock.AnyArg()).WillReturnError(sql.ErrNoRows)
			},
			user: &pb.User{
				Name:     "unregistered",
				Password: "123",
			},
			expected_err: auth.ErrUserNotExist,
		},
		{
			name: "wrong password",
			setup_mock: func(mock sqlmock.Sqlmock) {
				some_hash, _ := bcrypt.GenerateFromPassword([]byte("92JwfOyf3i8ibW9j/MuAIA=="), bcrypt.DefaultCost)
				rows := sqlmock.NewRows([]string{"password"}).AddRow(string(some_hash))
				mock.ExpectQuery("^SELECT password FROM users WHERE user = (.+)$").WithArgs(sqlmock.AnyArg()).WillReturnRows(rows)
			},
			user: &pb.User{
				Name:     "Mina",
				Password: "wrong password",
			},
			expected_err: auth.ErrWrongPassword,
		},
		{
			name: "normal login",
			setup_mock: func(mock sqlmock.Sqlmock) {
				hash, _ := bcrypt.GenerateFromPassword([]byte(GOOD_PASSWORD), bcrypt.DefaultCost)
				rows := sqlmock.NewRows([]string{"password"}).AddRow(string(hash))
				mock.ExpectQuery("^SELECT password FROM users WHERE user = (.+)$").WithArgs(sqlmock.AnyArg()).WillReturnRows(rows)
			},
			user: &pb.User{
				Name:     "Jason Statham",
				Password: GOOD_PASSWORD,
			},
			expected_err: nil,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			test.setup_mock(mock)

			token, err := client.Login(t.Context(), test.user)
			if !errors.Is(err, test.expected_err) {
				t.Errorf("expected error: %v, but got: %v", test.expected_err, err)
			}

			if test.expected_err != nil {
				return
			}

			if err := checkJWTUserMatch(test.user.Name, token.Token, JWT_SIGNATURE); err != nil {
				t.Error(err)
			}
		})
	}
}
