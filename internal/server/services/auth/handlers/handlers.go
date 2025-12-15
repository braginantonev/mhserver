package auth_handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/braginantonev/mhserver/pkg/auth"
	"github.com/braginantonev/mhserver/pkg/data"
	"github.com/braginantonev/mhserver/pkg/httperror"
)

func (handler Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		ErrFailedReadBody.Append(err).WithFuncName("LoginHandler.io.ReadAll").Write(w)
		return
	}

	if len(body) == 0 {
		ErrRequestBodyEmpty.Write(w)
		return
	}

	user := auth.User{}
	if err = json.Unmarshal(body, &user); err != nil {
		ErrBadJsonBody.Append(err).Write(w)
		return
	}

	slog.Info("Login request", slog.String("username", user.Name))

	token, err := auth.Login(user, handler.cfg.DB, handler.cfg.JWTSignature)
	if err != nil {
		handleServiceError(w, err, "auth.Login")
	} else {
		_, _ = w.Write([]byte(token))
	}
}

func (handler Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		ErrFailedReadBody.Append(err).WithFuncName("RegisterHandler.io.ReadAll").Write(w)
		return
	}

	if len(body) == 0 {
		ErrRequestBodyEmpty.Write(w)
		return
	}

	user := auth.User{}
	if err = json.Unmarshal(body, &user); err != nil {
		ErrBadJsonBody.Append(err).Write(w)
		return
	}

	slog.Info("Registration request", slog.String("username", user.Name))

	if err := auth.Register(user, handler.cfg.DB); err != nil {
		handleServiceError(w, err, "auth.Register")
		return
	}

	err = data.GenerateUserFolders(handler.cfg.WorkspacePath+user.Name, handler.cfg.UserCatalogs...)
	if err != nil {
		httperror.NewInternalHttpError(err, "RegisterHandler.data.GenerateUserFolders").Write(w)
	}
}
