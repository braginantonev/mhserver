package data_test

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/braginantonev/mhserver/pkg/data"
	pb "github.com/braginantonev/mhserver/proto/data"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const (
	TEST_WORKSPACE_PATH string = "/tmp/"
	TEST_USER           string = "TestUser"
	TEST_FILE           string = "test_file.txt"
	CHUNK_SIZE          int    = 75

	TEST_FILE_BODY string = `Антон Чигур никого не убивал!
Антон Чигур никого не убивал!
Антон Чигур никого не убивал!
Антон Чигур никого не убивал!
Антон Чигур никого не убивал!
Кто прочитал тот гей
`
)

func TestSaveData(t *testing.T) {
	grpc_server := grpc.NewServer()
	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), data.NewDataServerConfig(TEST_WORKSPACE_PATH, CHUNK_SIZE)))

	lis, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		panic(err)
	}
	defer lis.Close()

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

	test_file := strings.NewReader(TEST_FILE_BODY)

	//Todo: for files `file` type only
	err = os.MkdirAll(fmt.Sprintf("%s%s/files", TEST_WORKSPACE_PATH, TEST_USER), 0700)
	if err != nil {
		t.Fatal(err)
	}

	data_info := &pb.DataInfo{
		Type: pb.DataType_File,
		User: TEST_USER,
		File: TEST_FILE,
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

		<-time.After(150 * time.Millisecond)
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
	file, err := os.OpenFile(fmt.Sprintf("%s%s/files/%s", TEST_WORKSPACE_PATH, TEST_USER, TEST_FILE), os.O_RDONLY, 0660)
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
