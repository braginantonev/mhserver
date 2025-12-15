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
	slog.Info("Login request", slog.String("method", r.Method), slog.String("ip", r.RemoteAddr))

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		ErrFailedReadBody.Append(err).WithFuncName("Handlers.Login.io.ReadAll").Write(w)
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

	token, err := auth.Login(user, handler.cfg.DB, handler.cfg.JWTSignature)
	if err != nil {
		handleServiceError(w, err, "auth.Login")
	} else {
		_, _ = w.Write([]byte(token))
	}
}

func (handler Handler) Register(w http.ResponseWriter, r *http.Request) {
	slog.Info("Register request", slog.String("method", r.Method), slog.String("ip", r.RemoteAddr))

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		ErrFailedReadBody.Append(err).WithFuncName("Handlers.Register.io.ReadAll").Write(w)
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

	if err := auth.Register(user, handler.cfg.DB); err != nil {
		handleServiceError(w, err, "auth.Register")
		return
	}

	err = data.GenerateUserFolders(handler.cfg.WorkspacePath+user.Name, handler.cfg.UserCatalogs...)
	if err != nil {
		httperror.NewInternalHttpError(err, "Handlers.Register.data.GenerateUserFolders").Write(w)
	}
}
