package auth

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/braginantonev/mhserver/pkg/httperror"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	INSERT_USER   string = "INSERT INTO users (user, password) VALUES (?, ?)"
	SELECT_USERID string = "SELECT id FROM users WHERE user = ?"
	SELECT_USER   string = "SELECT user, password FROM users WHERE user = ?"
)

type User struct {
	Name     string `json:"user"`
	Password string `json:"pass"`
}

func NewUser(name string, password string) User {
	return User{
		Name:     name,
		Password: password,
	}
}

func Login(user User, db *sql.DB, jwt_signature string) (string, httperror.HttpError) {
	if user.Name == "" {
		return "", httperror.NewExternalHttpError(ErrNameIsEmpty, http.StatusBadRequest)
	}

	db_user := User{}
	row := db.QueryRow(SELECT_USER, user.Name)
	if err := row.Scan(&db_user.Name, &db_user.Password); err != nil {
		if err == sql.ErrNoRows {
			return "", httperror.NewExternalHttpError(ErrUserNotExist, http.StatusBadRequest)
		}

		return "", httperror.NewInternalHttpError(err, "Login")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(db_user.Password), []byte(user.Password)); err != nil {
		return "", httperror.NewExternalHttpError(ErrWrongPassword, http.StatusBadRequest)
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
		return "", httperror.NewInternalHttpError(err, "Login")
	}

	return token_str, httperror.NewEmptyHttpError()
}

func Register(user User, db *sql.DB) httperror.HttpError {
	if user.Name == "" {
		return httperror.NewExternalHttpError(ErrNameIsEmpty, http.StatusBadRequest)
	}

	row := db.QueryRow(SELECT_USERID, user.Name)
	if err := row.Scan(); err != sql.ErrNoRows {
		return httperror.NewExternalHttpError(ErrUserAlreadyExists, http.StatusContinue)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return httperror.NewInternalHttpError(err, "Register")
	}

	if _, err = db.Exec(INSERT_USER, user.Name, string(hash)); err != nil {
		return httperror.NewInternalHttpError(err, "Register")
	}

	return httperror.NewEmptyHttpError()
}

func CheckJWTUserMatch(username string, token string, signature string) error {
	tokenFromString, err := jwt.Parse(token, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(signature), nil
	})

	if err != nil {
		return fmt.Errorf("failed parse jwt: %s", err.Error())
	}

	if claims, ok := tokenFromString.Claims.(jwt.MapClaims); ok {
		if claims["name"] != username {
			return fmt.Errorf("expected user name: `%s`, but got `%s`", username, claims["name"])
		}
	} else {
		return fmt.Errorf("failed get claims from jwt")
	}

	return nil
}
