package data

import (
	data_handlers "github.com/braginantonev/mhserver/internal/http/data/handlers"
)

type DataService struct {
	Handler data_handlers.DataHandler
}

func NewDataService(handler data_handlers.DataHandler) *DataService {
	return &DataService{
		Handler: handler,
	}
}
