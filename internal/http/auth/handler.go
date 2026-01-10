package authhandler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	authconfig "github.com/braginantonev/mhserver/internal/config/auth"
	"github.com/braginantonev/mhserver/internal/grpc/data"
	"github.com/braginantonev/mhserver/internal/service/auth"
	"github.com/braginantonev/mhserver/pkg/httperror"
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
