package data

import "net/http"

type DataService interface {
	GetData(http.ResponseWriter, *http.Request)
	SaveData(http.ResponseWriter, *http.Request)
	GetHash(http.ResponseWriter, *http.Request)
}

//Todo: add handler
