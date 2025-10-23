package application

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/braginantonev/mhserver/pkg/auth"
	htypes "github.com/braginantonev/mhserver/pkg/handler_types"
)

//* --- LogReg --- *//

// If error is empty, return true
func log_error(w http.ResponseWriter, herr htypes.HandlerError, handler_name string) bool {
	switch herr.Type {
	case htypes.INTERNAL:
		slog.Error(herr.Error(), slog.String("handler", handler_name))
		w.WriteHeader(http.StatusInternalServerError)
		return false

	case htypes.EXTERNAL:
		w.WriteHeader(herr.Code)
		w.Write([]byte(fmt.Sprintf("error: %s", herr.Error())))
		return false
	}

	return true
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(body) == 0 {
		w.Write([]byte(MESSAGE_REQUEST_BODY_EMPTY))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user := auth.User{}
	if err = json.Unmarshal(body, &user); err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	slog.Info("Login request.", slog.String("username", user.Name))

	token, herr := auth.Login(user, DB, JWTSignature)
	if cont := log_error(w, herr, "login"); !cont {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(token))
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(body) == 0 {
		w.Write([]byte(MESSAGE_REQUEST_BODY_EMPTY))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user := auth.User{}
	if err = json.Unmarshal(body, &user); err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	slog.Info("Register request.", slog.String("username", user.Name))

	herr := auth.Register(user, DB)
	if cont := log_error(w, herr, "register"); !cont {
		return
	}

	w.WriteHeader(http.StatusOK)
}

//* --- Files --- *//

func DataHandler(get_fn http.HandlerFunc, save_fn http.HandlerFunc) http.HandlerFunc {
	fmt.Println("DataHandler mentioned")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Todo
	})
}

func GetDataHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("GetDataHandler mentioned")
	//Todo
}

func SaveDataHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("SaveDataHandler mentioned")
	//Todo
}

func GetHashHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("GetHashHandler mentioned")
	//Todo
}
