package auth_handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/braginantonev/mhserver/internal/server/services"
	"github.com/braginantonev/mhserver/pkg/auth"
)

func (handler Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(body) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(services.MESSAGE_REQUEST_BODY_EMPTY))
		return
	}

	user := auth.User{}
	if err = json.Unmarshal(body, &user); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	slog.Info("Login request.", slog.String("username", user.Name))

	token, herr := auth.Login(user, handler.cfg.DB, handler.cfg.JWTSignature)
	if cont := herr.Write(w); !cont {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(token))
}

func (handler Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(body) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(services.MESSAGE_REQUEST_BODY_EMPTY))
		return
	}

	user := auth.User{}
	if err = json.Unmarshal(body, &user); err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	slog.Info("Register request.", slog.String("username", user.Name))

	herr := auth.Register(user, handler.cfg.DB)
	if cont := herr.Write(w); !cont {
		return
	}

	w.WriteHeader(http.StatusOK)
}
