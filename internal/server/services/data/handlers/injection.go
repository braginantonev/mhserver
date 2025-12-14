package data_handlers

import (
	"net/http"

	"github.com/braginantonev/mhserver/pkg/data"
	pb "github.com/braginantonev/mhserver/proto/data"
)

type DataHandler interface {
	SaveData(w http.ResponseWriter, r *http.Request)
	GetData(w http.ResponseWriter, r *http.Request)
}

type Config struct {
	DataConfig        data.Config
	MaxRequestsCount  int
}

type Handler struct {
	cfg Config
	dataServiceClient pb.DataServiceClient
}

func NewDataHandler(cfg Config) Handler {
	return Handler{
		cfg: cfg,
	}
}
