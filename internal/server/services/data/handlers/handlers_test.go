package data_handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/braginantonev/mhserver/internal/server"
	data_handlers "github.com/braginantonev/mhserver/internal/server/services/data/handlers"
	"github.com/braginantonev/mhserver/pkg/data"
	"github.com/braginantonev/mhserver/pkg/httpcontextkeys"
	pb "github.com/braginantonev/mhserver/proto/data"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TestCase struct {
	name          string
	method        string
	data          *pb.Data
	expected_code int
	expected_body string
}

const (
	TEST_USERNAME string = "okabe"
	TEST_FILE     string = "cern_secrets.txt"
)

var (
	HandlerConfig = data_handlers.Config{
		DataConfig: data.Config{
			WorkspacePath: "/tmp/mhserver_tests/",
			ChunkSize:     25,
		},
		MaxRequestsCount: 25,
		// Data service client will be init
	}
	TestFileBody = []byte("hello world")
)

func createWorkdir(workspace_path, username string) error {
	return os.MkdirAll(fmt.Sprintf("%s%s/files", workspace_path, username), 0700)
}

func TestSaveData(t *testing.T) {
	err := createWorkdir(HandlerConfig.DataConfig.WorkspacePath, TEST_USERNAME)
	if err != nil {
		t.Fatal(err)
	}

	grpc_server := grpc.NewServer()
	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), HandlerConfig.DataConfig))

	lis, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := grpc_server.Serve(lis); err != nil {
			panic(err)
		}
	}()

	grpc_connection, err := grpc.NewClient("localhost:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}

	data_client := pb.NewDataServiceClient(grpc_connection)
	HandlerConfig.DataServiceClient = data_client

	handler := data_handlers.NewDataHandler(HandlerConfig)

	parallel_cases := []TestCase{
		{
			name:          "wrong method",
			method:        http.MethodDelete,
			data:          nil,
			expected_code: http.StatusMethodNotAllowed,
			expected_body: "",
		},
		{
			name:          "empty body",
			method:        http.MethodPost,
			data:          nil,
			expected_code: http.StatusBadRequest,
			expected_body: data_handlers.ErrRequestBodyEmpty.Error(),
		},
		{
			name:   "empty filename",
			method: http.MethodPost,
			data: &pb.Data{
				Info: &pb.DataInfo{
					File: "",
				},
				Part: &pb.FilePart{},
			},
			expected_code: http.StatusBadRequest,
			expected_body: data_handlers.ErrEmptyFilename.Error(),
		},
		{
			name:   "empty file part",
			method: http.MethodPatch,
			data: &pb.Data{
				Info: &pb.DataInfo{
					File: TEST_FILE,
				},
				Part: &pb.FilePart{},
			},
			expected_code: http.StatusBadRequest,
			expected_body: data_handlers.ErrEmptyFilePart.Error(),
		},
	}

	save_file_cases := []TestCase{
		{
			name:   "create file",
			method: http.MethodPost,
			data: &pb.Data{
				Info: &pb.DataInfo{
					Type: pb.DataType_File,
					File: TEST_FILE,
				},
				Part: &pb.FilePart{},
			},
			expected_code: http.StatusOK,
			expected_body: "",
		},
		{
			name:   "save data to file",
			method: http.MethodPatch,
			data: &pb.Data{
				Info: &pb.DataInfo{
					Type: pb.DataType_File,
					File: TEST_FILE,
				},
				Part: &pb.FilePart{
					Body:   TestFileBody,
					Offset: 0,
				},
			},
			expected_code: http.StatusOK,
			expected_body: "",
		},
		{
			name:   "finish rename file",
			method: http.MethodPut,
			data: &pb.Data{
				Info: &pb.DataInfo{
					Type: pb.DataType_File,
					File: TEST_FILE,
				},
				Part: &pb.FilePart{},
			},
			expected_code: http.StatusOK,
			expected_body: "",
		},
	}

	test_func := func(test TestCase, t *testing.T) {
		body := []byte("")
		if test.data != nil {
			body, err = json.Marshal(test.data)
			if err != nil {
				t.Fatal(err)
			}
		}

		req := httptest.NewRequest(test.method, server.LOGIN_ENDPOINT, bytes.NewReader(body))
		req = req.WithContext(context.WithValue(t.Context(), httpcontextkeys.USERNAME, TEST_USERNAME))
		w := httptest.NewRecorder()

		handler.SaveData(w, req)
		res := w.Result()
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != test.expected_code {
			t.Errorf("expected code %d, but got %d\ntest name: %s", test.expected_code, res.StatusCode, test.name)
		}

		got_body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		if string(got_body) != test.expected_body {
			t.Errorf("expected body: `%s`\nbut got: `%s`\ntest name: %s", test.expected_body, string(got_body), test.name)
		}
	}

	for _, test := range parallel_cases {
		t.Run(test.name, func(t *testing.T) {
			test_func(test, t)
		})
	}

	for _, test := range save_file_cases {
		test_func(test, t)
	}
}
