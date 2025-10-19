package application

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/braginantonev/mhserver/pkg/auth"
	types "github.com/braginantonev/mhserver/pkg/handler_types"
)

//* --- LogReg --- *//

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

	token, herr := auth.Login(user, DB, JWTSignature)
	if herr.Type != types.EMPTY {
		w.WriteHeader(herr.Code)
		w.Write([]byte(herr.Error()))
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

	if err := auth.Register(user, DB); err.Type != types.EMPTY {
		w.WriteHeader(err.Code)
		w.Write([]byte(err.Error()))
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
