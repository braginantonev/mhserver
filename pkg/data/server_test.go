package data_test

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/braginantonev/mhserver/pkg/data"
	pb "github.com/braginantonev/mhserver/proto/data"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const (
	WORKSPACE_PATH string = "/tmp/mhserver_tests/"
	TEST_USER      string = "user"
	CHUNK_SIZE     int    = 25

	TEST_FILE_BODY string = `Антон Чигур никого не убивал!
Антон Чигур никого не покарал!
Антон Чигур ничего не уничтожж!`
)

// Create server workspace in to test files with `File` type only
func createWorkspaceFolders(WorkspacePath, username string) error {
	return os.MkdirAll(fmt.Sprintf("%s%s/files", WorkspacePath, username), 0700)
}

func TestSaveData(t *testing.T) {
	test_file_name := "save_data_test_file.txt"

	if err := createWorkspaceFolders(WORKSPACE_PATH, TEST_USER); err != nil {
		t.Fatal(err)
	}

	grpc_server := grpc.NewServer()
	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), data.NewDataServerConfig(WORKSPACE_PATH, CHUNK_SIZE)))

	lis, err := net.Listen("tcp", "localhost:8081")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := grpc_server.Serve(lis); err != nil {
			panic(err)
		}
	}()

	grpc_connection, err := grpc.NewClient("localhost:8081", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}

	data_client := pb.NewDataServiceClient(grpc_connection)

	test_file := strings.NewReader(TEST_FILE_BODY)

	data_info := &pb.DataInfo{
		Type: pb.DataType_File,
		User: TEST_USER,
		File: test_file_name,
	}

	//* Test wrong action
	_, err = data_client.SaveData(t.Context(), &pb.Data{Action: pb.Action_Get, Info: data_info})
	if err != nil {
		s, _ := status.FromError(err)
		if s.Message() != data.ErrWrongAction.Error() {
			t.Errorf("expected error %s, but got %s", data.ErrWrongAction, s.Message())
		}
	}

	//* Normal saving
	// Create request
	_, err = data_client.SaveData(t.Context(), &pb.Data{
		Action: pb.Action_Create,
		Info:   data_info,
	})
	if err != nil {
		t.Error(err)
	}

	wg := sync.WaitGroup{}

	// Push file parts
	for i := 0; ; i++ {
		send_data := make([]byte, CHUNK_SIZE)
		n, err := test_file.Read(send_data)
		if err == io.EOF {
			break
		}

		wg.Add(1)
		go func(ch_id int, data []byte) {
			defer wg.Done()

			_, err = data_client.SaveData(t.Context(), &pb.Data{
				Action: pb.Action_Patch,
				Info:   data_info,
				Part: &pb.FilePart{
					Body:   data,
					Offset: int64(ch_id * CHUNK_SIZE),
				},
			})
			if err != nil {
				t.Error(err)
			}
		}(i, send_data[:n])
	}

	wg.Wait()

	// Finish request
	_, err = data_client.SaveData(t.Context(), &pb.Data{
		Action: pb.Action_Finish,
		Info:   data_info,
	})
	if err != nil {
		t.Error(err)
	}

	//! for files `file` type only
	file, err := os.OpenFile(fmt.Sprintf("%s%s/files/%s", WORKSPACE_PATH, TEST_USER, test_file_name), os.O_RDONLY, 0660)
	if err != nil {
		t.Fatal(err)
	}

	got_body_file, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}

	if string(got_body_file) != TEST_FILE_BODY {
		t.Error("got file body not implement expected")
	}
}

func TestGetData(t *testing.T) {
	test_file_name := "get_data_test_file.txt"

	if err := createWorkspaceFolders(WORKSPACE_PATH, TEST_USER); err != nil {
		t.Fatal(err)
	}

	// Create data grpc client
	grpc_server := grpc.NewServer()
	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), data.NewDataServerConfig(WORKSPACE_PATH, CHUNK_SIZE)))

	lis, err := net.Listen("tcp", "localhost:8082")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := grpc_server.Serve(lis); err != nil {
			panic(err)
		}
	}()

	grpc_connection, err := grpc.NewClient("localhost:8082", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}

	data_client := pb.NewDataServiceClient(grpc_connection)

	// Create test file
	file, err := os.OpenFile(fmt.Sprintf("%s%s/files/%s", WORKSPACE_PATH, TEST_USER, test_file_name), os.O_CREATE|os.O_RDWR, 0660)
	if err != nil {
		t.Fatal(err)
	}

	_, err = file.Write([]byte(TEST_FILE_BODY))
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()

	// Test
	got_file := bytes.NewBuffer(make([]byte, 0, len(TEST_FILE_BODY)))

	//! Real clients wants a speed, so in real request we use a parallels treads
	// len(TEST_FILE_BODY)/CHUNK_SIZE - imitation DataHandler.GetFileSize()
	for i := 0; i < 10; i++ {
		part, err := data_client.GetData(t.Context(), &pb.Data{
			Info: &pb.DataInfo{
				Type: pb.DataType_File,
				User: TEST_USER,
				File: test_file_name,
			},
			Action: pb.Action_Get,
			Part: &pb.FilePart{
				Offset: int64(i * CHUNK_SIZE),
			},
		})

		if err != nil {
			st, _ := status.FromError(err)
			if st.Message() != data.EOF.Error() {
				t.Error(st.Message())
			}
		}

		if _, loc_err := got_file.Write(part.GetBody()); loc_err != nil {
			t.Error(loc_err)
		}

		if part.IsLast {
			break
		}
	}

	if got_file.String() != TEST_FILE_BODY {
		t.Errorf("expected file body: `%s`\nbut got: `%s`", TEST_FILE_BODY, got_file.String())
	}
}
