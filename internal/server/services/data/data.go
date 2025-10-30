package data

import (
	"fmt"
	"net/http"
)

func GetData(w http.ResponseWriter, r *http.Request) {
	fmt.Println("GetDataHandler mentioned")
	//Todo
}

func SaveData(w http.ResponseWriter, r *http.Request) {
	fmt.Println("SaveDataHandler mentioned")
	//Todo
}

func GetHash(w http.ResponseWriter, r *http.Request) {
	fmt.Println("GetHashHandler mentioned")
	//Todo
}
