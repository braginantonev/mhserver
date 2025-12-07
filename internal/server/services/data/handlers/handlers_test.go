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

func testEmptyConnection(ctx context.Context, handler_func http.HandlerFunc, method, endpoint string) error {
	req := httptest.NewRequest(method, endpoint, nil)
	req = req.WithContext(context.WithValue(ctx, httpcontextkeys.USERNAME, TEST_USERNAME))
	w := httptest.NewRecorder()

	handler_func(w, req)
	res := w.Result()
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != data_handlers.ErrUnavailable.StatusCode {
		return fmt.Errorf("expected code %d, but got %d", data_handlers.ErrUnavailable.StatusCode, res.StatusCode)
	}

	got_body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if string(got_body) != data_handlers.ErrUnavailable.Error() {
		return fmt.Errorf("expected body: `%s`\nbut got: `%s`", data_handlers.ErrUnavailable.Error(), string(got_body))
	}

	return nil
}

func TestSaveData(t *testing.T) {
	hand_cfg := HandlerConfig

	err := createWorkdir(hand_cfg.DataConfig.WorkspacePath, TEST_USERNAME)
	if err != nil {
		t.Fatal(err)
	}

	grpc_server := grpc.NewServer()
	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), hand_cfg.DataConfig))

	lis, err := net.Listen("tcp", "localhost:8100")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := grpc_server.Serve(lis); err != nil {
			panic(err)
		}
	}()

	grpc_connection, err := grpc.NewClient("localhost:8100", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}

	data_client := pb.NewDataServiceClient(grpc_connection)

	// Test without connection to service
	err = testEmptyConnection(t.Context(), data_handlers.NewDataHandler(hand_cfg).SaveData, http.MethodPost, server.GET_DATA_ENDPOINT)
	if err != nil {
		t.Error(err)
	}

	hand_cfg.DataServiceClient = data_client
	handler := data_handlers.NewDataHandler(hand_cfg)

	filename := "test_save_data.txt"

	parallel_cases := [...]TestCase{
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
			name:   "empty file part",
			method: http.MethodPatch,
			data: &pb.Data{
				Info: &pb.DataInfo{
					File: filename,
				},
			},
			expected_code: http.StatusBadRequest,
			expected_body: data_handlers.ErrEmptyFilePart.Error(),
		},
	}

	save_file_cases := [...]TestCase{
		{
			name:   "create file",
			method: http.MethodPost,
			data: &pb.Data{
				Info: &pb.DataInfo{
					Type: pb.DataType_File,
					File: filename,
				},
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
					File: filename,
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
					File: filename,
				},
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

		req := httptest.NewRequest(test.method, server.SAVE_DATA_ENDPOINT, bytes.NewReader(body))
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

func TestGetData(t *testing.T) {
	hand_cfg := HandlerConfig

	err := createWorkdir(hand_cfg.DataConfig.WorkspacePath, TEST_USERNAME)
	if err != nil {
		t.Fatal(err)
	}

	grpc_server := grpc.NewServer()
	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), hand_cfg.DataConfig))

	lis, err := net.Listen("tcp", "localhost:8101")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := grpc_server.Serve(lis); err != nil {
			panic(err)
		}
	}()

	grpc_connection, err := grpc.NewClient("localhost:8101", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}

	data_client := pb.NewDataServiceClient(grpc_connection)

	// Test without connection to service
	err = testEmptyConnection(t.Context(), data_handlers.NewDataHandler(hand_cfg).GetData, http.MethodGet, server.SAVE_DATA_ENDPOINT)
	if err != nil {
		t.Error(err)
	}

	hand_cfg.DataServiceClient = data_client
	handler := data_handlers.NewDataHandler(hand_cfg)

	// Create test file
	filename := "test_get_data_handler.txt"
	file, err := os.OpenFile(fmt.Sprintf("%s%s/files/%s", hand_cfg.DataConfig.WorkspacePath, TEST_USERNAME, filename), os.O_CREATE|os.O_RDWR, 0660)
	if err != nil {
		t.Fatal(err)
	}

	_, err = file.Write(TestFileBody)
	if err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		TestCase
		body_is_json bool // If true - convert TestFileBody to JSON
	}{
		{
			TestCase: TestCase{
				name:          "wrong method",
				method:        http.MethodDelete,
				data:          nil,
				expected_code: http.StatusMethodNotAllowed,
			},
		},
		{
			TestCase: TestCase{
				name:          "empty body",
				method:        http.MethodGet,
				data:          nil,
				expected_code: http.StatusBadRequest,
				expected_body: data_handlers.ErrRequestBodyEmpty.Error(),
			},
		},
		{
			TestCase: TestCase{
				name:   "good get",
				method: http.MethodGet,
				data: &pb.Data{
					Info: &pb.DataInfo{
						File: filename,
					},
					Part: &pb.FilePart{
						Offset: 0,
					},
				},
				expected_code: http.StatusOK,
			},
			body_is_json: true,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			body := []byte("")
			if test.data != nil {
				body, err = json.Marshal(test.data)
				if err != nil {
					t.Fatal(err)
				}
			}

			req := httptest.NewRequest(test.method, server.GET_DATA_ENDPOINT, bytes.NewReader(body))
			req = req.WithContext(context.WithValue(t.Context(), httpcontextkeys.USERNAME, TEST_USERNAME))
			w := httptest.NewRecorder()

			handler.GetData(w, req)
			res := w.Result()
			defer func() { _ = res.Body.Close() }()

			if res.StatusCode != test.expected_code {
				t.Errorf("expected code %d, but got %d", test.expected_code, res.StatusCode)
			}

			if test.body_is_json {
				parsed, err := json.Marshal(pb.FilePart{
					Body:   TestFileBody,
					IsLast: true,
				})
				if err != nil {
					t.Error(err)
				}
				test.expected_body = string(parsed)
			}

			got_body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}

			if string(got_body) != test.expected_body {
				t.Errorf("expected body: `%s`\nbut got: `%s`", test.expected_body, string(got_body))
			}
		})
	}
}
