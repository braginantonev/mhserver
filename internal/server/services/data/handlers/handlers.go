package data_handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/braginantonev/mhserver/pkg/data"
	"github.com/braginantonev/mhserver/pkg/httpcontextkeys"
	"github.com/braginantonev/mhserver/pkg/httperror"
	pb "github.com/braginantonev/mhserver/proto/data"
)

var (
	saveActions = map[string]pb.Action{
		http.MethodPatch: pb.Action_Patch,
		http.MethodPost:  pb.Action_Create,
		http.MethodPut:   pb.Action_Finish,
	}
)

// Use only with auth_middlewares.WithAuth()
func (h Handler) SaveData(w http.ResponseWriter, r *http.Request) {
	action, ok := saveActions[r.Method]
	if !ok {
		w.WriteHeader(http.StatusMethodNotAllowed)
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

	ctx := context.Background()

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

	if r.Method == http.MethodPatch && len(save_data.GetPart().Body) == 0 {
		ErrEmptyFilePart.Write(w)
		return
	}

	_, err = h.cfg.DataServiceClient.SaveData(ctx, save_data)
	if err != nil && !errors.Is(err, data.EOF) {
		if errors.Is(errors.Unwrap(err), data.ErrInternal) {
			httperror.NewInternalHttpError(err, "Handlers.SaveData.SaveData").Write(w)
		}
		httperror.NewExternalHttpError(err, http.StatusBadRequest).Write(w)
	}
}

func (s Handler) GetData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
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

	ctx := context.Background()

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

	part, err := s.cfg.DataServiceClient.GetData(ctx, req_data)
	if err != nil {
		if errors.Is(errors.Unwrap(err), data.ErrInternal) {
			ErrInternal.Append(err).WithFuncName("Handlers.GetData.SaveData").Write(w)
		} else {
			httperror.NewExternalHttpError(err, http.StatusBadRequest).Write(w)
		}
		return
	}

	json_part, err := json.Marshal(part)
	if err != nil {
		ErrInternal.Append(err).WithFuncName("Handlers.GetData.Marshal").Write(w)
		return
	}

	w.Write(json_part)
}
