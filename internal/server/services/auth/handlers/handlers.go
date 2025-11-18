package auth_handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/braginantonev/mhserver/internal/server/services"
	"github.com/braginantonev/mhserver/pkg/auth"
	"github.com/braginantonev/mhserver/pkg/httperror"
)

func (handler Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		httperror.NewInternalHttpError(err, "LoginHandler.io.ReadAll").Write(w)
		return
	}

	if len(body) == 0 {
		httperror.NewExternalHttpError(services.ErrRequestBodyEmpty, http.StatusBadRequest).Write(w)
		return
	}

	user := auth.User{}
	if err = json.Unmarshal(body, &user); err != nil {
		httperror.NewExternalHttpError(err, http.StatusBadRequest).Write(w)
		return
	}

	slog.Info("Login request", slog.String("username", user.Name))

	token, err := auth.Login(user, handler.cfg.DB, handler.cfg.JWTSignature)
	if err != nil {
		writeError(w, err, "auth.Login")
	} else {
		services.WriteResponse(w, []byte(token), http.StatusOK)
	}
}

func (handler Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		httperror.NewInternalHttpError(err, "RegisterHandler.io.ReadAll").Write(w)
		return
	}

	if len(body) == 0 {
		httperror.NewExternalHttpError(services.ErrRequestBodyEmpty, http.StatusBadRequest).Write(w)
		return
	}

	user := auth.User{}
	if err = json.Unmarshal(body, &user); err != nil {
		httperror.NewInternalHttpError(err, "RegisterHandler.json.Unmarshal").Write(w)
		return
	}

	slog.Info("Register request.", slog.String("username", user.Name))

	if err := auth.Register(user, handler.cfg.DB); err != nil {
		writeError(w, err, "auth.Register")
	}
}
