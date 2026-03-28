package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	USER_NAME_MAX_LENGTH           int    = 30
	INSERT_USER                    string = "INSERT INTO users (user, password) VALUES (?, ?)"
	SELECT_USERID                  string = "SELECT id FROM users WHERE user = ?"
	SELECT_USER                    string = "SELECT user, password FROM users WHERE user = ?"
	SELECT_REGISTER_SECRET_KEY     string = "SELECT id FROM register_secret_keys WHERE secret_key = ?"
	DELETE_REGISTRATION_SECRET_KEY string = "DELETE FROM register_secret_keys WHERE id = ?"
)

type User struct {
	Name     string `json:"username"`
	Password string `json:"password"`
}

func NewUser(name string, password string) User {
	return User{
		Name:     name,
		Password: password,
	}
}

type RegisterUser struct {
	User
	Key string `json:"key"`
}

func NewRegisterUser(user User, key string) RegisterUser {
	return RegisterUser{
		User: user,
		Key:  key,
	}
}

// If user exist in database, return personal jwt token
func Login(user User, db *sql.DB, jwt_signature string) (string, error) {
	db_user := User{}
	row := db.QueryRow(SELECT_USER, user.Name)
	if err := row.Scan(&db_user.Name, &db_user.Password); err != nil {
		if err == sql.ErrNoRows {
			return "", ErrUserNotExist
		}

		slog.Error("failed scan sql rows", slog.Any("err", err))
		return "", ErrInternal
	}

	if err := bcrypt.CompareHashAndPassword([]byte(db_user.Password), []byte(user.Password)); err != nil {
		return "", ErrWrongPassword
	}

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"name": user.Name,
		"nbf":  now.Unix(),
		"exp":  now.Add(24 * time.Hour).Unix(),
		"iat":  now.Unix(),
	})

	token_str, err := token.SignedString([]byte(jwt_signature))
	if err != nil {
		slog.Error("failed complete signed jwt token", slog.Any("err", err))
		return "", ErrInternal
	}

	return token_str, nil
}

// Crypt user password and put them to database
func Register(user RegisterUser, db *sql.DB) error {
	if len(user.Name) > USER_NAME_MAX_LENGTH {
		return ErrNameTooLong
	}

	row := db.QueryRow(SELECT_USERID, user.Name)
	if err := row.Scan(); err != sql.ErrNoRows {
		return ErrUserAlreadyExists
	}

	var key_id int
	key_row := db.QueryRow(SELECT_REGISTER_SECRET_KEY, user.Key)
	if err := key_row.Scan(&key_id); errors.Is(err, sql.ErrNoRows) {
		return ErrRegSecretKeyNotFound
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed generate hash from password", slog.Any("err", err))
		return ErrInternal
	}

	if _, err = db.Exec(INSERT_USER, user.Name, string(hash)); err != nil {
		slog.Error("failed insert user to sql", slog.Any("err", err))
		return ErrInternal
	}

	if _, err = db.Exec(DELETE_REGISTRATION_SECRET_KEY, key_id); err != nil {
		slog.Error("failed delete registration secret key from sql", slog.Any("err", err))
		return ErrInternal
	}

	return nil
}

func CheckJWTUserMatch(username string, token string, signature string) error {
	tokenFromString, err := jwt.Parse(token, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %s", ErrJwtSignatureInvalid, token.Header["alg"])
		}

		return []byte(signature), nil
	})

	if err != nil {
		return err
	}

	if claims, ok := tokenFromString.Claims.(jwt.MapClaims); ok {
		if claims["name"] != username {
			return fmt.Errorf("%w\n\texpected user name: `%s`, but got `%s`", ErrWrongJWTName, username, claims["name"])
		}
	} else {
		return ErrBadClaims
	}

	return nil
}
