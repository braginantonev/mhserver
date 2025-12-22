package data_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"testing"

	dataconfig "github.com/braginantonev/mhserver/internal/config/data"
	"github.com/braginantonev/mhserver/internal/grpc/data"
	pb "github.com/braginantonev/mhserver/proto/data"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const (
	WORKSPACE_PATH string = "/tmp/mhserver_tests/"
	TEST_USER      string = "user"
	CHUNK_SIZE     int    = 1024

	TEST_FILE_BODY string = `Антон Чигур никого не убивал!
Антон Чигур никого не покарал!
Антон Чигур ничего не уничтожж!`
)

// Create server workspace in to test files with `File` type only
func createWorkspaceFolders(WorkspacePath, username string) error {
	return os.MkdirAll(fmt.Sprintf("%s%s/files", WorkspacePath, username), 0700)
}

// Client simulation
func saveFile(ctx context.Context, data_client pb.DataServiceClient, data_info *pb.DataInfo, reader io.Reader) error {
	// Create request
	_, err := data_client.SaveData(ctx, &pb.Data{
		Action: pb.Action_Create,
		Info:   data_info,
	})
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	errs := make(chan error, 1)

	// Push file parts
	for i := 0; ; i++ {
		send_data := make([]byte, data_info.GetSize().Chunk)
		n, err := reader.Read(send_data)
		if err == io.EOF {
			break
		}

		wg.Add(1)
		func(ch_id int, data []byte) {
			defer wg.Done()

			_, err = data_client.SaveData(ctx, &pb.Data{
				Action: pb.Action_Patch,
				Info:   data_info,
				Part: &pb.FilePart{
					Body:   data,
					Offset: int64(ch_id * int(data_info.GetSize().Chunk)),
				},
			})

			if err != nil {
				errs <- err
			}
		}(i, send_data[:n])
	}

	wg.Wait()
	close(errs)

	err = <-errs
	if err != nil {
		return err
	}

	// Finish request
	_, err = data_client.SaveData(ctx, &pb.Data{
		Action: pb.Action_Finish,
		Info:   data_info,
	})
	if err != nil {
		return fmt.Errorf("failed finish save file: %w", err)
	}

	return nil
}

func getRPCErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	st, ok := status.FromError(err)
	if ok {
		return st.Message()
	} else {
		return err.Error()
	}
}

func TestSaveData(t *testing.T) {
	test_file_name := "save_data_test_file.txt"

	if err := createWorkspaceFolders(WORKSPACE_PATH, TEST_USER); err != nil {
		t.Fatal(err)
	}

	grpc_server := grpc.NewServer()
	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), dataconfig.NewDataServerConfig(WORKSPACE_PATH, dataconfig.DataMemoryConfig{
		MaxChunkSize: 512 * 1024 * 1024,
		MinChunkSize: 4 * 1024,
		AvailableRAM: 1024 * 1024 * 1024,
	})))

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

	data_info := &pb.DataInfo{
		Type: pb.DataType_File,
		User: TEST_USER,
		File: test_file_name,
		Size: &pb.FileSize{
			Chunk: uint64(CHUNK_SIZE),
		},
	}

	//* Test wrong action
	_, err = data_client.SaveData(t.Context(), &pb.Data{Action: pb.Action_Get, Info: data_info})
	if err != nil {
		s, _ := status.FromError(err)
		if s.Message() != data.ErrWrongAction.Error() {
			t.Errorf("expected error %s, but got %s", data.ErrWrongAction, s.Message())
		}
	}

	err = saveFile(t.Context(), data_client, data_info, strings.NewReader(TEST_FILE_BODY))
	if err != nil {
		t.Fatal(err)
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
	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), dataconfig.NewDataServerConfig(WORKSPACE_PATH, dataconfig.DataMemoryConfig{
		MaxChunkSize: 512 * 1024 * 1024,
		MinChunkSize: 4 * 1024,
		AvailableRAM: 1024 * 1024 * 1024,
	})))

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
				Size: &pb.FileSize{
					Chunk: uint64(CHUNK_SIZE),
				},
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

func TestGetSum(t *testing.T) {
	if err := createWorkspaceFolders(WORKSPACE_PATH, TEST_USER); err != nil {
		t.Fatal(err)
	}

	// Create data grpc client
	grpc_server := grpc.NewServer()
	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), dataconfig.NewDataServerConfig(WORKSPACE_PATH, dataconfig.DataMemoryConfig{
		MaxChunkSize: 512 * 1024 * 1024,
		MinChunkSize: 4 * 1024,
		AvailableRAM: 1024 * 1024 * 1024,
	})))

	lis, err := net.Listen("tcp", "localhost:8083")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := grpc_server.Serve(lis); err != nil {
			panic(err)
		}
	}()

	grpc_connection, err := grpc.NewClient("localhost:8083", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}

	data_client := pb.NewDataServiceClient(grpc_connection)

	genRandomFile := func(ln uint64) string {
		var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ\n\t")
		letters_len := len(letters)

		result := make([]rune, ln)

		for i := range result {
			result[i] = letters[rand.Intn(letters_len)]
		}

		return string(result)
	}

	cases := [...]struct {
		name           string
		data_info      *pb.DataInfo
		file_body      string
		bad_sum_wanted bool
		expected_err   error
	}{
		{
			name: "empty file name",
			data_info: &pb.DataInfo{
				Type: pb.DataType_File,
				User: TEST_USER,
			},
			file_body:    TEST_FILE_BODY,
			expected_err: data.ErrEmptyFilename,
		},
		{
			name: "file 500 bytes",
			data_info: &pb.DataInfo{
				Type: pb.DataType_File,
				User: TEST_USER,
				File: "500b.txt",
				Size: &pb.FileSize{
					Chunk: 500 / 50,
				},
			},
			file_body:    genRandomFile(500),
			expected_err: fmt.Errorf(""),
		},
		{
			name: "file 10 kb",
			data_info: &pb.DataInfo{
				Type: pb.DataType_File,
				User: TEST_USER,
				File: "10kb.txt",
				Size: &pb.FileSize{
					Chunk: 10 * 1024 / 50,
				},
			},
			file_body:    genRandomFile(10 * 1024),
			expected_err: fmt.Errorf(""),
		},
		{
			name: "file 500 kb",
			data_info: &pb.DataInfo{
				Type: pb.DataType_File,
				User: TEST_USER,
				File: "500kb.txt",
				Size: &pb.FileSize{
					Chunk: 500 * 1024 / 50,
				},
			},
			file_body:    genRandomFile(500 * 1024),
			expected_err: fmt.Errorf(""),
		},
		{
			name: "file 5 mb",
			data_info: &pb.DataInfo{
				Type: pb.DataType_File,
				User: TEST_USER,
				File: "5mb.txt",
				Size: &pb.FileSize{
					Chunk: 5 * 1024 * 1024 / 50,
				},
			},
			file_body:    genRandomFile(5 * 1024 * 1024),
			expected_err: fmt.Errorf(""),
		},
		{
			name: "file 50 mb",
			data_info: &pb.DataInfo{
				Type: pb.DataType_File,
				User: TEST_USER,
				File: "50mb.txt",
				Size: &pb.FileSize{
					Chunk: 1024 * 1024,
				},
			},
			file_body:    genRandomFile(50 * 1024 * 1024),
			expected_err: fmt.Errorf(""),
		},
		{
			name: "file 100mb",
			data_info: &pb.DataInfo{
				Type: pb.DataType_File,
				User: TEST_USER,
				File: "100mb.txt",
				Size: &pb.FileSize{
					Chunk: 2 * 1024 * 1024,
				},
			},
			file_body:    genRandomFile(100 * 1024 * 1024),
			expected_err: fmt.Errorf(""),
		},
		{
			name: "bad client sum",
			data_info: &pb.DataInfo{
				Type: pb.DataType_File,
				User: TEST_USER,
				File: "bad_sum.txt",
				Size: &pb.FileSize{
					Chunk: 25,
				},
			},
			bad_sum_wanted: true,
			file_body:      genRandomFile(500),
			expected_err:   fmt.Errorf(""),
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			expected_sum := sha256.Sum256([]byte(test.file_body))
			if test.bad_sum_wanted {
				expected_sum[0] = 0
			}

			if err := saveFile(t.Context(), data_client, test.data_info, strings.NewReader(test.file_body)); err != nil {
				if mess := getRPCErrorMessage(err); mess != test.expected_err.Error() {
					t.Fatalf("expected error %s, but got %s", test.expected_err, mess)
				}
			}

			defer func(test_name string) {
				if test.data_info.File == "" {
					return
				}

				if err := os.Remove(fmt.Sprintf("%s%s/files/%s", WORKSPACE_PATH, test.data_info.User, test.data_info.File)); err != nil {
					t.Logf("failed remove file; test name = %s", test_name)
				}
			}(test.name)

			got_sum, err := data_client.GetSum(t.Context(), test.data_info)

			if mess := getRPCErrorMessage(err); mess != test.expected_err.Error() {
				t.Fatalf("expected error %s, but got %s", test.expected_err, mess)
			}

			if err != nil {
				return
			}

			for i, n := range got_sum.Sum {
				if n != expected_sum[i] && !test.bad_sum_wanted {
					t.Logf("expected last 250 bytes %s", test.file_body[len(test.file_body)-250:])
					t.Fatalf("expected sum: %x, but got: %x", expected_sum, got_sum.Sum)
				}
			}
		})
	}
}
