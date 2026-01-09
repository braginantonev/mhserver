package datahandler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/braginantonev/mhserver/pkg/httpcontextkeys"
	"github.com/braginantonev/mhserver/pkg/httpjsonutils"
	pb "github.com/braginantonev/mhserver/proto/data"
)

var (
	RequestTimeout = 5 * time.Second
)

type Handler struct {
	dataServiceClient pb.DataServiceClient
}

func NewDataHandler(grpc_client pb.DataServiceClient) Handler {
	return Handler{
		dataServiceClient: grpc_client,
	}
}

// Use only with auth_middlewares.WithAuth()
func (h Handler) CreateConnection(w http.ResponseWriter, r *http.Request) {
	slog.Info("Create connection request", slog.String("method", r.Method), slog.String("ip", r.RemoteAddr))

	if r.Method != http.MethodOptions {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Add("Content-Type", "text/plain")

	if h.dataServiceClient == nil {
		ErrUnavailable.Write(w)
		return
	}

	var req_info pb.DataInfo
	if err := httpjsonutils.ConvertJsonToStruct(&req_info, r.Body, "Handlers.SaveData"); err.StatusCode != 0 {
		err.Write(w)
		return
	}

	username, ok := r.Context().Value(httpcontextkeys.USERNAME).(string)
	if !ok {
		ErrWrongContextUsername.WithFuncName("Handlers.SaveData").Write(w)
		return
	}
	req_info.Username = username

	ctx, cancel := context.WithTimeout(context.Background(), RequestTimeout)
	defer cancel()

	conn, err := h.dataServiceClient.CreateConnection(ctx, &req_info)
	if err != nil {
		handleServiceError(err, w, "data.SaveData")
	}

	json_conn, err := json.Marshal(conn)
	if err != nil {
		ErrInternal.Append(err).WithFuncName("Handlers.CreateConnection.Marshal").Write(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(json_conn)
}

// Use only with auth_middlewares.WithAuth()
func (h Handler) SaveData(w http.ResponseWriter, r *http.Request) {
	slog.Info("Save data request", slog.String("method", r.Method), slog.String("ip", r.RemoteAddr))

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Add("Content-Type", "text/plain")

	if h.dataServiceClient == nil {
		ErrUnavailable.Write(w)
		return
	}

	var save_chunk pb.SaveChunk
	if err := httpjsonutils.ConvertJsonToStruct(&save_chunk, r.Body, "Handlers.SaveData"); err.StatusCode != 0 {
		err.Write(w)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), RequestTimeout)
	defer cancel()

	_, err := h.dataServiceClient.SaveData(ctx, &save_chunk)
	if err != nil {
		handleServiceError(err, w, "data.SaveData")
	}
}

func (h Handler) GetData(w http.ResponseWriter, r *http.Request) {
	slog.Info("Get data request", slog.String("method", r.Method), slog.String("ip", r.RemoteAddr))

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Add("Content-Type", "text/plain")

	if h.dataServiceClient == nil {
		ErrUnavailable.Write(w)
		return
	}

	var req_chunk pb.GetChunk
	if err := httpjsonutils.ConvertJsonToStruct(&req_chunk, r.Body, "Handlers.GetData"); err.StatusCode != 0 {
		err.Write(w)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), RequestTimeout)
	defer cancel()

	part, err := h.dataServiceClient.GetData(ctx, &req_chunk)
	if err != nil {
		handleServiceError(err, w, "data.GetData")
		return
	}

	json_part, err := json.Marshal(part)
	if err != nil {
		ErrInternal.Append(err).WithFuncName("Handlers.GetData.Marshal").Write(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(json_part)
}

func (h Handler) GetSum(w http.ResponseWriter, r *http.Request) {
	slog.Info("Get sum request", slog.String("method", r.Method), slog.String("ip", r.RemoteAddr))

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Add("Content-Type", "text/plain")

	if h.dataServiceClient == nil {
		ErrUnavailable.Write(w)
		return
	}

	var get_chunk pb.GetChunk
	if err := httpjsonutils.ConvertJsonToStruct(&get_chunk, r.Body, "Handlers.GetSum"); err.StatusCode != 0 {
		err.Write(w)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), RequestTimeout)
	defer cancel()

	sum, err := h.dataServiceClient.GetSum(ctx, &get_chunk)
	if err != nil {
		handleServiceError(err, w, "data.GetSum")
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")

	_, _ = w.Write(sum.Sum)
}
