package data_handlers

import (
	"net/http"

	"github.com/braginantonev/mhserver/pkg/data"
	pb "github.com/braginantonev/mhserver/proto/data"
)

type DataHandler interface {
	SaveData(http.ResponseWriter, *http.Request)
	GetData(http.ResponseWriter, *http.Request)
	GetSum(http.ResponseWriter, *http.Request)
}

type Config struct {
	DataConfig       data.Config
	MaxRequestsCount int
}

type Handler struct {
	cfg               Config
	dataServiceClient pb.DataServiceClient
}

func NewDataHandler(cfg Config, grpc_client pb.DataServiceClient) Handler {
	return Handler{
		cfg:               cfg,
		dataServiceClient: grpc_client,
	}
}
