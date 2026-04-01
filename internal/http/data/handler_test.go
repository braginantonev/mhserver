package datahttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	dataconfig "github.com/braginantonev/mhserver/internal/config/data"
	"github.com/braginantonev/mhserver/internal/grpc/data"
	datahttp "github.com/braginantonev/mhserver/internal/http/data"
	"github.com/braginantonev/mhserver/internal/server"
	"github.com/braginantonev/mhserver/pkg/httpcontextkeys"
	"github.com/braginantonev/mhserver/pkg/httpjsonutils"
	pb "github.com/braginantonev/mhserver/proto/data"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TestCase struct {
	name                  string
	method                string
	expected_content_type string
	expected_code         int
	expected_body         string
}

const (
	TEST_WORKSPACE_PATH string = "/tmp/mhserver_tests/"
	TEST_USERNAME       string = "okabe"
	TEST_FILE_BODY      string = "hello world!"
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

	if res.StatusCode != datahttp.ErrUnavailable.Status() {
		return fmt.Errorf("expected code %d, but got %d", datahttp.ErrUnavailable.Status(), res.StatusCode)
	}

	got_body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if string(got_body) != datahttp.ErrUnavailable.Description() {
		return fmt.Errorf("expected body: `%s`\nbut got: `%s`", datahttp.ErrUnavailable.Description(), string(got_body))
	}

	return nil
}

func TestSaveDataHandler(t *testing.T) {
	err := createWorkdir(TEST_WORKSPACE_PATH, TEST_USERNAME)
	if err != nil {
		t.Fatal(err)
	}

	grpc_server := grpc.NewServer()
	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), dataconfig.NewDataServerConfig(TEST_WORKSPACE_PATH, dataconfig.DataMemoryConfig{
		MaxChunkSize: 512 * 1024 * 1024,
		MinChunkSize: 4 * 1024,
		AvailableRAM: 1024 * 1024 * 1024,
	})))

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
	t.Run("service unavailable", func(t *testing.T) {
		err = testEmptyConnection(t.Context(), datahttp.NewDataHandler(nil).SaveData, http.MethodPost, server.SAVE_DATA_ENDPOINT)
		if err != nil {
			t.Error(err)
		}
	})

	handler := datahttp.NewDataHandler(data_client)

	cases := [...]struct {
		TestCase
		directory string
		filename  string
		save_body []byte
		body_len  uint64
	}{
		{
			TestCase: TestCase{
				name:                  "empty body",
				method:                http.MethodPost,
				expected_code:         http.StatusBadRequest,
				expected_content_type: "text/plain",
				expected_body:         httpjsonutils.ErrRequestBodyEmpty.Description(),
			},
			directory: "/",
			filename:  "sht normal save.txt",
			body_len:  5,
		},
		{
			TestCase: TestCase{
				name:          "normal save",
				method:        http.MethodPost,
				expected_code: http.StatusOK,
			},
			directory: "/",
			filename:  "sht normal save.txt",
			save_body: []byte(TEST_FILE_BODY),
			body_len:  uint64(len(TEST_FILE_BODY)),
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			conn, err := data_client.CreateConnection(t.Context(), &pb.ConnectionRequest{
				Mode:      pb.ConnectionMode_RDWR,
				Username:  TEST_USERNAME,
				Directory: test.directory,
				Filename:  test.filename,
				Size:      test.body_len,
			})
			if err != nil {
				t.Fatalf("failed create connection; err: %v", err)
			}

			body := []byte("")
			if test.save_body != nil {
				body, err = json.Marshal(pb.FilePart{Chunk: test.save_body})
				if err != nil {
					t.Fatal(err)
				}
			}

			req := httptest.NewRequest(test.method, fmt.Sprintf("%s?uuid=%s", server.SAVE_DATA_ENDPOINT, conn.UUID), bytes.NewReader(body))
			req = req.WithContext(context.WithValue(t.Context(), httpcontextkeys.USERNAME, TEST_USERNAME))
			w := httptest.NewRecorder()

			handler.SaveData(w, req)
			res := w.Result()
			defer func() { _ = res.Body.Close() }()

			if res.StatusCode != test.expected_code {
				t.Errorf("expected code %d, but got %d\ntest name: %s", test.expected_code, res.StatusCode, test.name)
			}

			content_type := res.Header.Get("Content-Type")
			if !strings.Contains(content_type, test.expected_content_type) {
				t.Errorf("expected content-type `%s`, but got `%s`", test.expected_content_type, content_type)
			}

			got_body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}

			if string(got_body) != test.expected_body {
				t.Errorf("expected body: `%s`\nbut got: `%s`\ntest name: %s", test.expected_body, string(got_body), test.name)
			}
		})
	}
}

func TestGetDataHandler(t *testing.T) {
	err := createWorkdir(TEST_WORKSPACE_PATH, TEST_USERNAME)
	if err != nil {
		t.Fatal(err)
	}

	grpc_server := grpc.NewServer()
	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), dataconfig.NewDataServerConfig(TEST_WORKSPACE_PATH, dataconfig.DataMemoryConfig{
		MaxChunkSize: 512 * 1024 * 1024,
		MinChunkSize: 4 * 1024,
		AvailableRAM: 1024 * 1024 * 1024,
	})))

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
	t.Run("service unavailable", func(t *testing.T) {
		err = testEmptyConnection(t.Context(), datahttp.NewDataHandler(nil).GetData, http.MethodGet, server.GET_DATA_ENDPOINT)
		if err != nil {
			t.Error(err)
		}
	})

	handler := datahttp.NewDataHandler(data_client)

	// Create test file

	test_dir := "/"
	test_file := "test_get_data_handler.txt"
	file, err := os.OpenFile(fmt.Sprintf("%s%s/files%s%s", TEST_WORKSPACE_PATH, TEST_USERNAME, test_dir, test_file), os.O_CREATE|os.O_RDWR, 0660)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = os.Remove(fmt.Sprintf("%s%s/files%s%s", TEST_WORKSPACE_PATH, TEST_USERNAME, test_dir, test_file))
		if err != nil {
			slog.WarnContext(t.Context(), "failed remove test files", slog.Any("err", err))
		}
	}()

	_, err = file.Write([]byte(TEST_FILE_BODY))
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()

	cases := [...]TestCase{
		{
			// Compare with TEST_FILE_BODY
			name:                  "normal get",
			method:                http.MethodGet,
			expected_code:         http.StatusOK,
			expected_body:         TEST_FILE_BODY,
			expected_content_type: "application/octet-stream",
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			conn, err := data_client.CreateConnection(t.Context(), &pb.ConnectionRequest{
				Mode:      pb.ConnectionMode_RDONLY,
				Username:  TEST_USERNAME,
				Directory: test_dir,
				Filename:  test_file,
			})
			if err != nil {
				t.Fatalf("failed create connection; err: %v", err)
			}

			req := httptest.NewRequest(test.method, fmt.Sprintf("%s?uuid=%s&chunkID=%d", server.GET_DATA_ENDPOINT, conn.UUID, 0), nil)
			req = req.WithContext(context.WithValue(t.Context(), httpcontextkeys.USERNAME, TEST_USERNAME))
			w := httptest.NewRecorder()

			handler.GetData(w, req)
			res := w.Result()
			defer func() { _ = res.Body.Close() }()

			if res.StatusCode != test.expected_code {
				t.Errorf("expected code %d, but got %d", test.expected_code, res.StatusCode)
			}

			content_type := res.Header.Get("Content-Type")
			if !strings.Contains(content_type, test.expected_content_type) {
				t.Errorf("expected content-type `%s`, but got `%s`", test.expected_content_type, content_type)
			}

			got_body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}

			if content_type == "application/json" {
				var got_part struct {
					Chunk string `json:"chunk"`
				}

				err = json.Unmarshal(got_body, &got_part)
				if err != nil {
					t.Fatalf("bad got json; err: %v", err)
				}

				if got_part.Chunk != TEST_FILE_BODY {
					t.Errorf("expected chunk `%s`, but got `%s`", TEST_FILE_BODY, string(got_part.Chunk))
				}
			} else {
				if string(got_body) != test.expected_body {
					t.Errorf("expected body: `%s`\nbut got: `%s`", test.expected_body, string(got_body))
				}
			}
		})
	}
}
