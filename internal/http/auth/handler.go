package authhttp

import (
	"log/slog"
	"net/http"

	authconfig "github.com/braginantonev/mhserver/internal/config/auth"
	"github.com/braginantonev/mhserver/internal/repository/dirs"
	"github.com/braginantonev/mhserver/internal/service/auth"
	"github.com/braginantonev/mhserver/pkg/httpjsonutils"
)

type Handler struct {
	cfg authconfig.AuthHandlerConfig
}

func NewHandler(cfg authconfig.AuthHandlerConfig) Handler {
	return Handler{
		cfg: cfg,
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

	token, err := auth.Login(user, handler.cfg.DB, handler.cfg.JWTSignature)
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

	if err := auth.Register(user, handler.cfg.DB); err != nil {
		handleServiceError(w, err, "auth.Register")
		return
	}

	err := dirs.GenerateUserFolders(handler.cfg.WorkspacePath, user.Name, handler.cfg.UserCatalogs...)
	if err != nil {
		ErrInternal.Append(err).WithFuncName("Handlers.Register.dirs.GenerateUserFolders").Write(w)
	}

	w.Header().Del("Content-Type")
}
