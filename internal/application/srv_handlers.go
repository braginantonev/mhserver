package application

import (
	"fmt"
	"net/http"
)

//* --- LogReg --- *//

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("LoginHandler mentioned")
	//Todo
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("RegisterHandler mentioned")
	//Todo
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
