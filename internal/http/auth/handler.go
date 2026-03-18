package authhandler

import (
	"log/slog"
	"net/http"

	authconfig "github.com/braginantonev/mhserver/internal/config/auth"
	"github.com/braginantonev/mhserver/internal/grpc/data"
	"github.com/braginantonev/mhserver/internal/service/auth"
	"github.com/braginantonev/mhserver/pkg/httpjsonutils"
)

type Handler struct {
	cfg authconfig.AuthHandlerConfig
}

func NewAuthHandler(cfg authconfig.AuthHandlerConfig) Handler {
	return Handler{
		cfg: cfg,
	}
}

func (handler Handler) Login(w http.ResponseWriter, r *http.Request) {
	slog.Info("Login request", slog.String("method", r.Method), slog.String("ip", r.RemoteAddr))

	var user auth.User
	if err := httpjsonutils.ConvertJsonToStruct(&user, r.Body, "Handlers.Login"); err != nil {
		err.Write(w)
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

	var user auth.User
	if err := httpjsonutils.ConvertJsonToStruct(&user, r.Body, "Handlers.Register"); err != nil {
		err.Write(w)
		return
	}

	if err := auth.Register(user, handler.cfg.DB); err != nil {
		handleServiceError(w, err, "auth.Register")
		return
	}

	err := data.GenerateUserFolders(handler.cfg.WorkspacePath+user.Username, handler.cfg.UserCatalogs...)
	if err != nil {
		ErrInternal.Append(err).WithFuncName("Handlers.Register.data.GenerateUserFolders").Write(w)
	}
}
