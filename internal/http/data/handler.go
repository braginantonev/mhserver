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
	"github.com/gorilla/mux"
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

	conn, err := h.dataServiceClient.CreateConnection(r.Context(), &req_info)
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

	// If !ok use uuid from json. It's using for tests
	uuid, ok := mux.Vars(r)["uuid"]
	if ok {
		save_chunk.UUID = uuid
	}

	_, err := h.dataServiceClient.SaveData(r.Context(), &save_chunk)
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

	var get_chunk pb.GetChunk
	if ch_id := r.URL.Query().Get("chunkID"); ch_id != "" {
		res, err := strconv.Atoi(ch_id)
		if err != nil {
			ErrBadQuery.Write(w)
			return
		}
		get_chunk.ChunkId = int32(res)
	} else {
		if err := httpjsonutils.ConvertJsonToStruct(&get_chunk, r.Body, "Handlers.GetData"); err.StatusCode != 0 {
			err.Write(w)
			return
		}
	}

	// If !ok use uuid from json. It's using for tests
	uuid, ok := mux.Vars(r)["uuid"]
	if ok {
		get_chunk.UUID = uuid
	}

	part, err := h.dataServiceClient.GetData(r.Context(), &get_chunk)
	if err != nil {
		handleServiceError(err, w, "data.GetData")
		return
	}

	resp_file_part := struct {
		Chunk string `json:"chunk"`
	}{Chunk: string(part.Chunk)}

	json_part, err := json.Marshal(resp_file_part)
	if err != nil {
		ErrInternal.Append(err).WithFuncName("Handlers.GetData.Marshal").Write(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(json_part)
}

func (h Handler) GetSum(w http.ResponseWriter, r *http.Request) {
	slog.Info("Get sum request", slog.String("method", r.Method), slog.String("ip", r.RemoteAddr))

	w.Header().Add("Content-Type", "text/plain")

	if h.dataServiceClient == nil {
		ErrUnavailable.Write(w)
		return
	}

	var get_chunk pb.GetChunk
	if ch_id := r.URL.Query().Get("chunkID"); ch_id != "" {
		res, err := strconv.Atoi(ch_id)
		if err != nil {
			ErrBadQuery.Write(w)
			return
		}
		get_chunk.ChunkId = int32(res)
	} else {
		if err := httpjsonutils.ConvertJsonToStruct(&get_chunk, r.Body, "Handlers.GetData"); err.StatusCode != 0 {
			err.Write(w)
			return
		}
	}

	// If !ok use uuid from json. It's using for tests
	uuid, ok := mux.Vars(r)["uuid"]
	if ok {
		get_chunk.UUID = uuid
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
		ErrWrongContextUsername.WithFuncName("Handlers.SaveData").Write(w)
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

	resp, err := json.Marshal(files.Infos)
	if err != nil {
		ErrInternal.Append(err).WithFuncName("Handler.GetFiles.Marshal").Write(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(resp)
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
		ErrWrongContextUsername.WithFuncName("Handlers.SaveData").Write(w)
		return
	}

	resp, err := h.dataServiceClient.GetAvailableDiskSpace(r.Context(), &pb.Direction{
		User: username,
		Dir:  r.URL.Query().Get("dir"),
	})
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
		ErrWrongContextUsername.WithFuncName("Handlers.SaveData").Write(w)
		return
	}

	_, err := h.dataServiceClient.CreateDir(r.Context(), &pb.Direction{
		User: username,
		Dir:  r.URL.Query().Get("dir"),
	})
	if err != nil {
		handleServiceError(err, w, "data.GetAvailableDiskSpace")
		return
	}

	w.Header().Del("Content-Type")
	w.WriteHeader(http.StatusOK)
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
		ErrWrongContextUsername.WithFuncName("Handlers.SaveData").Write(w)
		return
	}

	_, err := h.dataServiceClient.RemoveDir(r.Context(), &pb.Direction{
		User: username,
		Dir:  r.URL.Query().Get("dir"),
	})
	if err != nil {
		handleServiceError(err, w, "data.GetAvailableDiskSpace")
		return
	}

	w.Header().Del("Content-Type")
	w.WriteHeader(http.StatusOK)
}
