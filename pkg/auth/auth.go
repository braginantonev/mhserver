package auth

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	types "github.com/braginantonev/mhserver/pkg/handler_types"
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

func Login(user User, db *sql.DB, jwt_signature string) (string, types.HandlerError) {
	if user.Name == "" {
		return "", types.NewExternalHandlerError(ErrNameIsEmpty, http.StatusBadRequest)
	}

	db_user := User{}
	row := db.QueryRow(SELECT_USER, user.Name)
	if err := row.Scan(&db_user.Name, &db_user.Password); err != nil {
		if err == sql.ErrNoRows {
			return "", types.NewExternalHandlerError(ErrUserNotExist, http.StatusNotFound)
		}

		return "", types.NewInternalHandlerError()
	}

	if err := bcrypt.CompareHashAndPassword([]byte(db_user.Password), []byte(user.Password)); err != nil {
		return "", types.NewExternalHandlerError(ErrWrongPassword, http.StatusBadRequest)
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
		slog.Error("In Login(): " + err.Error())
		return "", types.NewInternalHandlerError()
	}

	return token_str, types.NewEmptyHandlerError()
}

func Register(user User, db *sql.DB) types.HandlerError {
	if user.Name == "" {
		return types.NewExternalHandlerError(ErrNameIsEmpty, http.StatusBadRequest)
	}

	row := db.QueryRow(SELECT_USERID, user.Name)
	if err := row.Scan(); err != sql.ErrNoRows {
		return types.NewExternalHandlerError(ErrUserAlreadyExists, http.StatusContinue)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("In Register(): " + err.Error())
		return types.NewInternalHandlerError()
	}

	if _, err = db.Exec(INSERT_USER, user.Name, string(hash)); err != nil {
		slog.Error("In Register(): " + err.Error())
		return types.NewInternalHandlerError()
	}

	return types.NewEmptyHandlerError()
}
