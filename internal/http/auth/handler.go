package authhttp

import (
	"log/slog"
	"net/http"

	"github.com/braginantonev/mhserver/internal/service/auth"
	"github.com/braginantonev/mhserver/pkg/httpjsonutils"
)

type Handler struct {
	service *auth.AuthService
}

func NewHandler(service *auth.AuthService) Handler {
	return Handler{
		service,
	}
}

func (handler Handler) Login(w http.ResponseWriter, r *http.Request) {
	slog.Info("Login request", slog.String("method", r.Method), slog.String("ip", r.RemoteAddr))

	w.Header().Add("Content-Type", "text/plain")

	var user auth.User
	if err := httpjsonutils.ConvertJsonToStruct(&user, r.Body, "Handlers.Login"); err != nil {
		err.Write(w)
		return
	}

	if user.Name == "" {
		ErrUsernameEmpty.Write(w)
		return
	}

	token, err := handler.service.Login(user)
	if err != nil {
		handleServiceError(w, err, "auth.Login")
	} else {
		_, _ = w.Write([]byte(token))
	}

	w.Header().Del("Content-Type")
}

func (handler Handler) Register(w http.ResponseWriter, r *http.Request) {
	slog.Info("Register request", slog.String("method", r.Method), slog.String("ip", r.RemoteAddr))

	w.Header().Add("Content-Type", "text/plain")

	var user auth.RegisterUser
	if err := httpjsonutils.ConvertJsonToStruct(&user, r.Body, "Handlers.Register"); err != nil {
		err.Write(w)
		return
	}

	if user.Name == "" {
		ErrUsernameEmpty.Write(w)
		return
	}

	if user.Key == "" {
		ErrRegSecretKeyEmpty.Write(w)
		return
	}

	if err := handler.service.Register(user); err != nil {
		handleServiceError(w, err, "auth.Register")
		return
	}

	w.Header().Del("Content-Type")
}
