package data_handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/braginantonev/mhserver/pkg/httpcontextkeys"
	pb "github.com/braginantonev/mhserver/proto/data"
)

var (
	saveActions = map[string]pb.Action{
		http.MethodPatch: pb.Action_Patch,
		http.MethodPost:  pb.Action_Create,
		http.MethodPut:   pb.Action_Finish,
	}

	RequestTimeout = 5 * time.Second
)

// Use only with auth_middlewares.WithAuth()
func (h Handler) SaveData(w http.ResponseWriter, r *http.Request) {
	action, ok := saveActions[r.Method]
	if !ok {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if h.cfg.DataServiceClient == nil {
		ErrUnavailable.Write(w)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		ErrFailedReadBody.Append(err).WithFuncName("Handlers.SaveData.io.ReadAll").Write(w)
		return
	}

	if len(body) == 0 {
		ErrRequestBodyEmpty.Write(w)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), RequestTimeout)
	defer cancel()

	save_data := &pb.Data{}
	if err = json.Unmarshal(body, &save_data); err != nil {
		ErrBadJsonBody.Append(err).Write(w)
		return
	}

	save_data.Action = action

	username, ok := r.Context().Value(httpcontextkeys.USERNAME).(string)
	if !ok {
		ErrWrongContextUsername.WithFuncName("Handlers.SaveData").Write(w)
		return
	}
	save_data.Info.User = username

	if r.Method == http.MethodPatch && len(save_data.GetPart().GetBody()) == 0 {
		ErrEmptyFilePart.Write(w)
		return
	}

	_, err = h.cfg.DataServiceClient.SaveData(ctx, save_data)
	if err != nil {
		handleServiceError(err, w, "data.SaveData")
	}
}

func (h Handler) GetData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if h.cfg.DataServiceClient == nil {
		ErrUnavailable.Write(w)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		ErrFailedReadBody.Append(err).WithFuncName("Handlers.GetData.io.ReadAll").Write(w)
		return
	}

	if len(body) == 0 {
		ErrRequestBodyEmpty.Write(w)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), RequestTimeout)
	defer cancel()

	req_data := &pb.Data{}
	if err = json.Unmarshal(body, &req_data); err != nil {
		ErrBadJsonBody.Append(err).Write(w)
		return
	}

	req_data.Action = pb.Action_Get

	username, ok := r.Context().Value(httpcontextkeys.USERNAME).(string)
	if !ok {
		ErrWrongContextUsername.WithFuncName("Handlers.GetData").Write(w)
		return
	}
	req_data.Info.User = username

	part, err := h.cfg.DataServiceClient.GetData(ctx, req_data)
	if err != nil {
		handleServiceError(err, w, "data.GetData")
		return
	}

	json_part, err := json.Marshal(part)
	if err != nil {
		ErrInternal.Append(err).WithFuncName("Handlers.GetData.Marshal").Write(w)
		return
	}

	_, _ = w.Write(json_part)
}
