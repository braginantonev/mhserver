package datahandler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/braginantonev/mhserver/pkg/httpcontextkeys"
	"github.com/braginantonev/mhserver/pkg/httpjsonutils"
	pb "github.com/braginantonev/mhserver/proto/data"
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

	w.Header().Add("Content-Type", "text/plain")

	if h.dataServiceClient == nil {
		ErrUnavailable.Write(w)
		return
	}

	var req_info pb.DataInfo
	if err := httpjsonutils.ConvertJsonToStruct(&req_info, r.Body, "Handlers.CreateConnection"); err != nil {
		err.Write(w)
		return
	}

	req_info.Directory = r.URL.Query().Get("dir")

	username, ok := r.Context().Value(httpcontextkeys.USERNAME).(string)
	if !ok {
		ErrWrongContextUsername.WithFuncName("Handlers.CreateConnection").Write(w)
		return
	}
	req_info.Username = username

	conn, err := h.dataServiceClient.CreateConnection(r.Context(), &req_info)
	if err != nil {
		handleServiceError(err, w, "data.CreateConnection")
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(conn); err != nil {
		ErrInternal.Append(err).WithFuncName("Handlers.CreateConnection.Marshal").Write(w)
	}
}

// Use only with auth_middlewares.WithAuth()
func (h Handler) SaveData(w http.ResponseWriter, r *http.Request) {
	slog.Info("Save data request", slog.String("method", r.Method), slog.String("ip", r.RemoteAddr))

	w.Header().Add("Content-Type", "text/plain")

	if h.dataServiceClient == nil {
		ErrUnavailable.Write(w)
		return
	}

	var save_chunk pb.FilePart
	if err := httpjsonutils.ConvertJsonToStruct(&save_chunk, r.Body, "Handlers.SaveData"); err != nil {
		err.Write(w)
		return
	}

	_, err := h.dataServiceClient.SaveData(r.Context(), &pb.SaveChunk{
		UUID: r.URL.Query().Get("uuid"),
		Data: &save_chunk,
	})
	if err != nil {
		handleServiceError(err, w, "data.SaveData")
	}
}

func (h Handler) GetData(w http.ResponseWriter, r *http.Request) {
	slog.Info("Get data request", slog.String("method", r.Method), slog.String("ip", r.RemoteAddr))

	w.Header().Add("Content-Type", "text/plain")

	if h.dataServiceClient == nil {
		ErrUnavailable.Write(w)
		return
	}

	chunk_id, err := strconv.Atoi(r.URL.Query().Get("chunkID"))
	if err != nil {
		ErrBadUuidFormat.Write(w)
		return
	}

	get_chunk := pb.GetChunk{
		ChunkId: int32(chunk_id),
		UUID:    r.URL.Query().Get("uuid"),
	}

	part, err := h.dataServiceClient.GetData(r.Context(), &get_chunk)
	if err != nil {
		handleServiceError(err, w, "data.GetData")
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(part.Chunk)
}

func (h Handler) GetSum(w http.ResponseWriter, r *http.Request) {
	slog.Info("Get sum request", slog.String("method", r.Method), slog.String("ip", r.RemoteAddr))

	w.Header().Add("Content-Type", "text/plain")

	if h.dataServiceClient == nil {
		ErrUnavailable.Write(w)
		return
	}

	chunk_id, err := strconv.Atoi(r.URL.Query().Get("chunkID"))
	if err != nil {
		ErrBadUuidFormat.Write(w)
		return
	}

	get_chunk := pb.GetChunk{
		ChunkId: int32(chunk_id),
		UUID:    r.URL.Query().Get("uuid"),
	}

	sum, err := h.dataServiceClient.GetSum(r.Context(), &get_chunk)
	if err != nil {
		handleServiceError(err, w, "data.GetSum")
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(sum.Sum)
}

func (h Handler) GetFiles(w http.ResponseWriter, r *http.Request) {
	slog.Info("Get files list request", slog.String("method", r.Method), slog.String("ip", r.RemoteAddr))

	w.Header().Add("Content-Type", "text/plain")

	if h.dataServiceClient == nil {
		ErrUnavailable.Write(w)
		return
	}

	username, ok := r.Context().Value(httpcontextkeys.USERNAME).(string)
	if !ok {
		ErrWrongContextUsername.WithFuncName("Handlers.GetFiles").Write(w)
		return
	}

	files, err := h.dataServiceClient.GetFiles(r.Context(), &pb.Direction{
		User: username,
		Dir:  r.URL.Query().Get("dir"),
	})
	if err != nil {
		handleServiceError(err, w, "data.GetFiles")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(files.Infos); err != nil {
		ErrInternal.Append(err).WithFuncName("Handler.GetFiles.Marshal").Write(w)
	}
}

func (h Handler) GetAvailableDiskSpace(w http.ResponseWriter, r *http.Request) {
	slog.Info("Get available disk space request", slog.String("method", r.Method), slog.String("ip", r.RemoteAddr))

	w.Header().Add("Content-Type", "text/plain")

	if h.dataServiceClient == nil {
		ErrUnavailable.Write(w)
		return
	}

	username, ok := r.Context().Value(httpcontextkeys.USERNAME).(string)
	if !ok {
		ErrWrongContextUsername.WithFuncName("Handlers.GetAvailableDiskSpace").Write(w)
		return
	}

	resp, err := h.dataServiceClient.GetAvailableDiskSpace(r.Context(), &pb.Direction{User: username})
	if err != nil {
		handleServiceError(err, w, "data.GetAvailableDiskSpace")
		return
	}

	_, _ = fmt.Fprint(w, resp.Val)
}

func (h Handler) CreateDir(w http.ResponseWriter, r *http.Request) {
	slog.Info("Create dir request", slog.String("method", r.Method), slog.String("ip", r.RemoteAddr))

	w.Header().Add("Content-Type", "text/plain")

	if h.dataServiceClient == nil {
		ErrUnavailable.Write(w)
		return
	}

	username, ok := r.Context().Value(httpcontextkeys.USERNAME).(string)
	if !ok {
		ErrWrongContextUsername.WithFuncName("Handlers.CreateDir").Write(w)
		return
	}

	_, err := h.dataServiceClient.CreateDir(r.Context(), &pb.Direction{
		User: username,
		Dir:  r.URL.Query().Get("dir"),
	})
	if err != nil {
		handleServiceError(err, w, "data.CreateDir")
		return
	}

	w.Header().Del("Content-Type")
}

func (h Handler) RemoveDir(w http.ResponseWriter, r *http.Request) {
	slog.Info("Remove dir request", slog.String("method", r.Method), slog.String("ip", r.RemoteAddr))

	w.Header().Add("Content-Type", "text/plain")

	if h.dataServiceClient == nil {
		ErrUnavailable.Write(w)
		return
	}

	username, ok := r.Context().Value(httpcontextkeys.USERNAME).(string)
	if !ok {
		ErrWrongContextUsername.WithFuncName("Handlers.RemoveDir").Write(w)
		return
	}

	_, err := h.dataServiceClient.RemoveDir(r.Context(), &pb.Direction{
		User: username,
		Dir:  r.URL.Query().Get("dir"),
	})
	if err != nil {
		handleServiceError(err, w, "data.RemoveDir")
		return
	}

	w.Header().Del("Content-Type")
}
