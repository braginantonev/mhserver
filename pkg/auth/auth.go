package auth

import (
	"database/sql"
	"log/slog"
	"net/http"

	types "github.com/braginantonev/mhserver/pkg/handler_types"
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

func Login(user User, db *sql.DB) (string, error) {
	return "", nil
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
		slog.Error(err.Error())
		return types.NewInternalHandlerError()
	}

	if _, err = db.Exec(INSERT_USER, user.Name, string(hash)); err != nil {
		slog.Error(err.Error())
		return types.NewInternalHandlerError()
	}

	return types.NewEmptyHandlerError()
}
