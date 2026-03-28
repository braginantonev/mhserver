package auth_test

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/braginantonev/mhserver/internal/repository/database"
	"github.com/braginantonev/mhserver/internal/service/auth"
	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

const (
	TEST_REGISTER_SECRET_KEY   string = "TEST_SECRET_KEY"
	INSERT_REGISTER_SECRET_KEY string = "INSERT INTO register_secret_keys (secret_key) VALUES (?)"
)

func InsertRegisterKeyToDB(db *sql.DB, secret_key string) error {
	_, err := db.Exec(INSERT_REGISTER_SECRET_KEY, secret_key)
	return err
}

func TestRegister(t *testing.T) {
	cases := [...]struct {
		name         string
		user         auth.RegisterUser
		get_from_db  bool // Check user field in db
		expected_err error
	}{
		{
			name:         "Empty name",
			user:         auth.NewRegisterUser(auth.NewUser("", "123"), ""),
			expected_err: auth.ErrNameIsEmpty,
		},
		{
			name:         "long username",
			user:         auth.NewRegisterUser(auth.NewUser("[Cop Killers] X1_BestCockSucker_1X", "123"), ""),
			expected_err: auth.ErrNameTooLong,
		},
		{
			name:         "key not found",
			user:         auth.NewRegisterUser(auth.NewUser("without reg", "123"), "WRONG KEY"),
			expected_err: auth.ErrRegSecretKeyNotFound,
		},
		{
			name:         "Base register",
			user:         auth.NewRegisterUser(auth.NewUser("register_test1", "123"), TEST_REGISTER_SECRET_KEY),
			get_from_db:  true,
			expected_err: nil,
		},
		{
			name:         "Already register",
			user:         auth.NewRegisterUser(auth.NewUser("register_test2", "123"), ""),
			expected_err: auth.ErrUserAlreadyExists,
			get_from_db:  true,
		},
	}

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
			if test.user.Key == TEST_REGISTER_SECRET_KEY {
				if err := InsertRegisterKeyToDB(db, TEST_REGISTER_SECRET_KEY); err != nil {
					t.Fatalf("failed to insert register key to DB: %v", err)
				}
			}

			err := auth.Register(test.user, db)

			if !errors.Is(err, test.expected_err) {
				t.Errorf("expected error: %s, but got: %s", test.expected_err, err)
			}

			if !test.get_from_db {
				return
			}

			db_user := auth.User{}
			row := db.QueryRow(auth.SELECT_USER, test.user.Name)

			if err = row.Scan(&db_user.Name, &db_user.Password); err != nil {
				t.Error(err)
			}

			if db_user.Name != test.user.Name {
				t.Errorf("expected name %s, but got %s", test.user.Name, db_user.Name)
			}

			if err = bcrypt.CompareHashAndPassword([]byte(db_user.Password), []byte(test.user.Password)); err != nil {
				t.Log(db_user.Password)
				t.Errorf("password incorrect. error=%s", err.Error())
			}

			row = db.QueryRow(auth.SELECT_REGISTER_SECRET_KEY, test.user.Key)
			if err := row.Scan(); !errors.Is(err, sql.ErrNoRows) {
				t.Error("secret key not deleted after registration")
			}
		})

		_, err := db.Exec("DELETE FROM users WHERE user = ?", test.user.Name)
		if err != nil {
			fmt.Println(err)
		}
	}
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

	jwt_signature := "test"
	registered_user := auth.NewRegisterUser(auth.NewUser("test_login1", "123"), TEST_REGISTER_SECRET_KEY)

	if err := InsertRegisterKeyToDB(db, TEST_REGISTER_SECRET_KEY); err != nil {
		t.Fatalf("failed to insert register key to DB: %v", err)
	}

	if err := auth.Register(registered_user, db); err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name         string
		user         auth.User
		expected_err error
		check_reg    bool
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
			user:         auth.NewUser(registered_user.Name, "WRONG"),
			expected_err: auth.ErrWrongPassword,
		},
		{
			name:         "Normal login",
			user:         auth.NewUser(registered_user.Name, "123"),
			expected_err: nil,
			check_reg:    true,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			token, err := auth.Login(test.user, db, jwt_signature)
			if !errors.Is(err, test.expected_err) {
				t.Errorf("expected error: %v, but got: %v", test.expected_err, err)
			}

			if !test.check_reg {
				return
			}

			if err := auth.CheckJWTUserMatch(test.user.Name, token, jwt_signature); err != nil {
				t.Error(err)
			}
		})
	}

	_, err = db.Exec("DELETE FROM users WHERE user = ?", registered_user.Name)
	if err != nil {
		fmt.Println(err)
	}
}
