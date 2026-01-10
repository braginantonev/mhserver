package datahandler_test

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
	"strings"
	"testing"

	dataconfig "github.com/braginantonev/mhserver/internal/config/data"
	"github.com/braginantonev/mhserver/internal/grpc/data"
	datahandler "github.com/braginantonev/mhserver/internal/http/data"
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

	if res.StatusCode != datahandler.ErrUnavailable.StatusCode {
		return fmt.Errorf("expected code %d, but got %d", datahandler.ErrUnavailable.StatusCode, res.StatusCode)
	}

	got_body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if string(got_body) != datahandler.ErrUnavailable.Error() {
		return fmt.Errorf("expected body: `%s`\nbut got: `%s`", datahandler.ErrUnavailable.Error(), string(got_body))
	}

	return nil
}

func TestSaveData(t *testing.T) {
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
		err = testEmptyConnection(t.Context(), datahandler.NewDataHandler(nil).SaveData, http.MethodPost, server.SAVE_DATA_ENDPOINT)
		if err != nil {
			t.Error(err)
		}
	})

	handler := datahandler.NewDataHandler(data_client)

	cases := [...]struct {
		TestCase
		filename  string
		save_body []byte
	}{
		{
			TestCase: TestCase{
				name:                  "empty body",
				method:                http.MethodPost,
				expected_code:         http.StatusBadRequest,
				expected_content_type: "text/plain",
				expected_body:         httpjsonutils.ErrRequestBodyEmpty.Error(),
			},
			filename: "sht normal save.txt",
		},
		{
			TestCase: TestCase{
				name:          "normal save",
				method:        http.MethodPost,
				expected_code: http.StatusOK,
			},
			filename:  "sht normal save.txt",
			save_body: []byte(TEST_FILE_BODY),
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			conn, err := data_client.CreateConnection(t.Context(), &pb.DataInfo{
				Username: TEST_USERNAME,
				Filename: test.filename,
				Filetype: pb.FileType_File,
				Size:     uint64(len(test.save_body)),
			})
			if err != nil {
				t.Fatalf("failed create connection; err: %v", err)
			}

			body := []byte("")
			if test.save_body != nil {
				body, err = json.Marshal(pb.SaveChunk{
					UUID: conn.UUID,
					Data: &pb.FilePart{
						Chunk: test.save_body,
					},
				})
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

func TestGetData(t *testing.T) {
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
		err = testEmptyConnection(t.Context(), datahandler.NewDataHandler(nil).GetData, http.MethodGet, server.GET_DATA_ENDPOINT)
		if err != nil {
			t.Error(err)
		}
	})

	handler := datahandler.NewDataHandler(data_client)

	// Create test file
	filename := "test_get_data_handler.txt"
	file, err := os.OpenFile(fmt.Sprintf("%s%s/files/%s", TEST_WORKSPACE_PATH, TEST_USERNAME, filename), os.O_CREATE|os.O_RDWR, 0660)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = file.Close()
		_ = os.Remove(fmt.Sprintf("%s%s/files/%s", TEST_WORKSPACE_PATH, TEST_USERNAME, filename))
	}()

	_, err = file.Write([]byte(TEST_FILE_BODY))
	if err != nil {
		t.Fatal(err)
	}

	cases := [...]TestCase{
		{
			name:                  "empty body",
			method:                http.MethodGet,
			expected_code:         http.StatusBadRequest,
			expected_body:         httpjsonutils.ErrRequestBodyEmpty.Error(),
			expected_content_type: "text/plain",
		},
		{
			// Compare with TEST_FILE_BODY
			name:                  "normal get",
			method:                http.MethodGet,
			expected_code:         http.StatusOK,
			expected_content_type: "application/json",
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			conn, err := data_client.CreateConnection(t.Context(), &pb.DataInfo{
				Username: TEST_USERNAME,
				Filename: filename,
				Filetype: pb.FileType_File,
				Size:     uint64(len(TEST_FILE_BODY)),
			})
			if err != nil {
				t.Fatalf("failed create connection; err: %v", err)
			}

			body := []byte("")
			if test.expected_body != httpjsonutils.ErrRequestBodyEmpty.Error() {
				body, err = json.Marshal(pb.GetChunk{
					UUID:    conn.UUID,
					ChunkId: 0,
				})
				if err != nil {
					t.Fatalf("failed marshal pb.GetChunk; err: %v", err)
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

			content_type := res.Header.Get("Content-Type")
			if !strings.Contains(content_type, test.expected_content_type) {
				t.Errorf("expected content-type `%s`, but got `%s`", test.expected_content_type, content_type)
			}

			got_body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}

			if content_type == "application/json" {
				var got_part pb.FilePart
				err = json.Unmarshal(got_body, &got_part)
				if err != nil {
					t.Errorf("bad got json; err: %v", err)
				}

				if string(got_part.Chunk) != TEST_FILE_BODY {
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
