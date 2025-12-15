package domain

import "net/http"

type DataHandler interface {
	SaveData(http.ResponseWriter, *http.Request)
	GetData(http.ResponseWriter, *http.Request)
	GetSum(http.ResponseWriter, *http.Request)
}

type HttpDataService struct {
	Handler DataHandler
}

func NewDataService(handler DataHandler) *HttpDataService {
	return &HttpDataService{
		Handler: handler,
	}
}
