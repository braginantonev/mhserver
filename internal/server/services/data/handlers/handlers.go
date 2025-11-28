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
	methodsActions = map[string]pb.Action{
		http.MethodPatch: pb.Action_Patch,
		http.MethodPost:  pb.Action_Create,
		http.MethodPut:   pb.Action_Finish,
	}
)

// Use only with auth_middlewares.WithAuth()
func (h Handler) SaveData(w http.ResponseWriter, r *http.Request) {
	action, ok := methodsActions[r.Method]
	if !ok {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		ErrFailedReadBody.Append(err).WithFuncName("Handler.GetData.io.ReadAll").Write(w)
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
		ErrWrongContextUsername.WithFuncName("Handler.GetData").Write(w)
		return
	}

	save_data.Info.User = username

	if save_data.GetInfo().File == "" {
		ErrEmptyFilename.Write(w)
		return
	}

	if r.Method == http.MethodPatch && len(save_data.GetPart().Body) == 0 {
		ErrEmptyFilePart.Write(w)
		return
	}

	_, err = h.cfg.DataServiceClient.SaveData(ctx, save_data)
	if err != nil && !errors.Is(err, data.EOF) {
		if errors.Is(errors.Unwrap(err), data.ErrInternal) {
			httperror.NewInternalHttpError(err, "Handler.GetData.SaveData").Write(w)
		}
		httperror.NewExternalHttpError(err, http.StatusBadRequest).Write(w)
	}
}
