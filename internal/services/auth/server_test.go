package auth_test

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/braginantonev/mhserver/internal/repository/database"
	"github.com/braginantonev/mhserver/internal/services"
	"github.com/braginantonev/mhserver/internal/services/auth"
	pb "github.com/braginantonev/mhserver/proto/gen/auth"
	"github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	TEST_REGISTER_SECRET_KEY   string = "TEST_SECRET_KEY"
	INSERT_REGISTER_SECRET_KEY string = "INSERT INTO register_secret_keys (secret_key) VALUES (?)"

	JWT_SIGNATURE  string = "123zxc"
	WORKSPACE_PATH string = "/tmp/mhserver_tests"
)

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

// return cleanup function or error
func insertTempSecretKey(db *sql.DB, secret_key string) (error, func()) {
	if _, err := db.Exec(INSERT_REGISTER_SECRET_KEY, secret_key); err != nil {
		return err, nil
	}

	return nil, func() {
		_, _ = db.Exec("DELETE FROM register_secret_keys WHERE secret_key = ?", secret_key)
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

	// Create data grpc client
	grpc_server := grpc.NewServer()
	pb.RegisterAuthServiceServer(grpc_server, auth.NewAuthServer(auth.NewAuthServiceConfig(WORKSPACE_PATH, JWT_SIGNATURE), db))

	lis, err := net.Listen("tcp", "localhost:8085")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := grpc_server.Serve(lis); err != nil {
			panic(err)
		}
	}()

	grpc_connection, err := grpc.NewClient("localhost:8085", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}

	service_client := pb.NewAuthServiceClient(grpc_connection)

	cases := [...]struct {
		name         string
		reg_info     *pb.RegisterRequest
		expected_err error
	}{
		{
			name: "long username",
			reg_info: &pb.RegisterRequest{
				User: &pb.User{
					Name:     "[Cop Killers] X1_BestCockSucker_1X",
					Password: "123",
				},
				SecretKey: "",
			},
			expected_err: auth.ErrNameTooLong,
		},
		{
			name: "key not found",
			reg_info: &pb.RegisterRequest{
				User: &pb.User{
					Name:     "unregistered",
					Password: "123",
				},
				SecretKey: "WRONG KEY",
			},
			expected_err: auth.ErrRegSecretKeyNotFound,
		},
		{
			name: "base register",
			reg_info: &pb.RegisterRequest{
				User: &pb.User{
					Name:     "register_test1",
					Password: "123",
				},
				SecretKey: TEST_REGISTER_SECRET_KEY,
			},
			expected_err: nil,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if test.reg_info.SecretKey == TEST_REGISTER_SECRET_KEY {
				err, clean := insertTempSecretKey(db, TEST_REGISTER_SECRET_KEY)
				if err != nil {
					t.Fatalf("failed to insert register key to DB: %v", err)
				}
				defer clean()
			}

			_, err := service_client.Register(t.Context(), test.reg_info)
			if !services.IsFromGRPC(err, test.expected_err) {
				t.Errorf("expected error: %s, but got: %s", test.expected_err, err)
			}

			// skip check in database
			if test.expected_err != nil {
				return
			}

			var pass_from_db string
			if err = db.QueryRow(auth.SELECT_USER_PASS, test.reg_info.User.Name).Scan(&pass_from_db); err != nil {
				t.Error(err)
			}

			if err = bcrypt.CompareHashAndPassword([]byte(pass_from_db), []byte(test.reg_info.User.Password)); err != nil {
				t.Errorf("password incorrect. error=%s", err.Error())
			}

			var reg_key int
			if err = db.QueryRow(auth.SELECT_REGISTER_SECRET_KEY, test.reg_info.SecretKey).Scan(&reg_key); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return
				}
				t.Error("failed check delete reg key.", err.Error())
			} else {
				t.Errorf("secret key not deleted after registration (key_id = %d)", reg_key)
			}
		})

		_, err := db.Exec("DELETE FROM users WHERE user = ?", test.reg_info.User.Name)
		if err != nil {
			fmt.Println(err)
		}
	}

	t.Run("already registered", func(t *testing.T) {
		username := "register_test2"
		err, clean := insertTempSecretKey(db, TEST_REGISTER_SECRET_KEY)
		if err != nil {
			t.Fatalf("failed to insert register key to DB: %v", err)
		}
		defer clean()

		// register user
		hash, err := bcrypt.GenerateFromPassword([]byte("123"), bcrypt.DefaultCost)
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.Exec(auth.INSERT_REG_USER, username, string(hash))
		if err != nil {
			t.Fatal(err)
		}

		if _, err = service_client.Register(t.Context(), &pb.RegisterRequest{
			User: &pb.User{
				Name:     username,
				Password: "123",
			},
			SecretKey: "xzc",
		}); !services.IsFromGRPC(err, auth.ErrUserAlreadyExists) {
			t.Errorf("expected error `%s`, but got `%s`", auth.ErrUserAlreadyExists, err)
		}

		// cleanup
		if _, err = db.Exec("DELETE FROM users WHERE user = ?", username); err != nil {
			t.Errorf("failed delete user (err = %s)", err)
		}
	})

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

	// Create data grpc client
	grpc_server := grpc.NewServer()
	pb.RegisterAuthServiceServer(grpc_server, auth.NewAuthServer(auth.NewAuthServiceConfig(WORKSPACE_PATH, JWT_SIGNATURE), db))

	lis, err := net.Listen("tcp", "localhost:8085")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := grpc_server.Serve(lis); err != nil {
			panic(err)
		}
	}()

	grpc_connection, err := grpc.NewClient("localhost:8085", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}

	service_client := pb.NewAuthServiceClient(grpc_connection)

	// register user

	registered_user := &pb.User{
		Name:     "login_user_1",
		Password: "123",
	}

	err, clean := insertTempSecretKey(db, TEST_REGISTER_SECRET_KEY)
	if err != nil {
		t.Fatalf("failed insert register key (err = %s)", err)
	}
	defer clean()

	if _, err := service_client.Register(t.Context(), &pb.RegisterRequest{
		User:      registered_user,
		SecretKey: TEST_REGISTER_SECRET_KEY,
	}); err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name         string
		user         *pb.User
		expected_err error
		check_reg    bool
	}{
		{
			name: "not registered",
			user: &pb.User{
				Name:     "unregistered user",
				Password: "123",
			},
			expected_err: auth.ErrUserNotExist,
		},
		{
			name: "wrong password",
			user: &pb.User{
				Name:     registered_user.Name,
				Password: "wrong password",
			},
			expected_err: auth.ErrWrongPassword,
		},
		{
			name: "normal login",
			user: &pb.User{
				Name:     registered_user.Name,
				Password: registered_user.Password,
			},
			expected_err: nil,
			check_reg:    true,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			token, err := service_client.Login(t.Context(), test.user)
			if !services.IsFromGRPC(err, test.expected_err) {
				t.Errorf("expected error: %v, but got: %v", test.expected_err, err)
			}

			if !test.check_reg {
				return
			}

			if err := checkJWTUserMatch(test.user.Name, token.Token, JWT_SIGNATURE); err != nil {
				t.Error(err)
			}
		})
	}

	_, err = db.Exec("DELETE FROM users WHERE user = ?", registered_user.Name)
	if err != nil {
		fmt.Println(err)
	}
}
