package data

import (
	"fmt"
	"net/http"
)

type DataService interface {
	GetData(http.ResponseWriter, *http.Request)
	SaveData(http.ResponseWriter, *http.Request)
	GetHash(http.ResponseWriter, *http.Request)
}

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
