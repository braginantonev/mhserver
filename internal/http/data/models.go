package datahttp

import "net/http"

type DataHandler interface {
	CreateConnection(http.ResponseWriter, *http.Request)
	SaveData(http.ResponseWriter, *http.Request)
	GetData(http.ResponseWriter, *http.Request)
	GetSum(http.ResponseWriter, *http.Request)
	GetFiles(http.ResponseWriter, *http.Request)
	GetAvailableDiskSpace(http.ResponseWriter, *http.Request)
	CreateDir(http.ResponseWriter, *http.Request)
	RemoveDir(http.ResponseWriter, *http.Request)
}

type DataMiddleware interface {
	WithRateLimit(http.HandlerFunc) http.HandlerFunc
}

type DataTransport struct {
	DataHandler
	DataMiddleware
}

func NewDataTransport(handler DataHandler, middleware DataMiddleware) *DataTransport {
	return &DataTransport{
		DataHandler:    handler,
		DataMiddleware: middleware,
	}
}
