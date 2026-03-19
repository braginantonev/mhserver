package data_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"testing"

	dataconfig "github.com/braginantonev/mhserver/internal/config/data"
	"github.com/braginantonev/mhserver/internal/grpc/data"
	pb "github.com/braginantonev/mhserver/proto/data"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const (
	WORKSPACE_PATH string = "/tmp/mhserver_tests/"
	TEST_USER      string = "user"
	CHUNK_SIZE     int    = 1024

	TEST_FILE_BODY string = `- Скажи, дружище, ты стихи любишь?
	- Стихи? Ну, не особо, сэр.
	- Тихо в лесу, только не спит только медведь... Он ещё с вечера начал пердеть. Вот и не спит медведь.
	...
	- Тихо в лесу, только не спит ёж. Нюхает ёж медвежий пердёжь, вот и не спит ёж.
	- Эм... Что?
	- Тихо в лесу, только не спит сова. Есть у совы смешная трава, вот и не спит сова.
	- Сэр, может, заправку закончим? `
)

// Create server workspace in to test files with `File` type only
func createWorkspaceFolders(workspace_path, username string) error {
	return os.MkdirAll(fmt.Sprintf("%s%s/files", workspace_path, username), 0700)
}

// Client simulation
func saveFile(ctx context.Context, data_client pb.DataServiceClient, req *pb.ConnectionRequest, reader io.Reader) error {
	conn, err := data_client.CreateConnection(ctx, req)
	if err != nil {
		return err
	}

	log.Printf("save file (%s) with size %d; chunk size = %d, count = %d", req.Filename, req.Size, conn.ChunkSize, conn.ChunksCount)

	wg := sync.WaitGroup{}
	errs := make(chan error, 1)

	// Push file parts
	for i := 0; ; i++ {
		send_data := make([]byte, conn.ChunkSize)
		n, err := reader.Read(send_data)
		if err == io.EOF {
			break
		}

		wg.Add(1)
		go func(ch_id int, data []byte) {
			defer wg.Done()

			_, err = data_client.SaveData(ctx, &pb.SaveChunk{
				UUID: conn.UUID,
				Data: &pb.FilePart{
					Chunk:  data,
					Offset: int64(ch_id * int(conn.ChunkSize)),
				},
			})

			if err != nil {
				errs <- err
			}
		}(i, send_data[:n])
	}

	wg.Wait()
	close(errs)

	return <-errs
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
	if err := createWorkspaceFolders(WORKSPACE_PATH, TEST_USER); err != nil {
		t.Fatal(err)
	}

	grpc_server := grpc.NewServer()
	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), dataconfig.NewDataServerConfig(WORKSPACE_PATH, dataconfig.DataMemoryConfig{
		MaxChunkSize: 25,                 //byte
		MinChunkSize: 5,                  //byte
		AvailableRAM: 1024 * 1024 * 1024, //byte
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

	t.Run("save without connection", func(t *testing.T) {
		random_uuid := uuid.New()
		_, err := data_client.SaveData(t.Context(), &pb.SaveChunk{
			UUID: random_uuid.String(),
			Data: &pb.FilePart{
				Chunk: []byte("be be be"),
			},
		})

		if m := getRPCErrorMessage(err); m != data.ErrConnectionNotFound.Error() {
			t.Errorf("expected error %s, but got %s", data.ErrConnectionNotFound.Error(), m)
		}
	})

	// To test: "save in test dir"
	test_dir := "/test_dir/"
	if err = os.MkdirAll(fmt.Sprintf("%s%s/files%s", WORKSPACE_PATH, TEST_USER, test_dir), 0700); err != nil {
		t.Fatal(err)
	}

	small_test_file := "I use arch btw"
	small_test_file_len := uint64(len(small_test_file))

	cases := [...]struct {
		name         string
		conn_info    *pb.ConnectionRequest
		save_data    string
		expected_err error
	}{
		{
			name: "save in root dir",
			conn_info: &pb.ConnectionRequest{
				Username:  TEST_USER,
				Mode:      pb.ConnectionMode_RDWR,
				Directory: "/",
				Filename:  "save_data_single.txt",
				Filetype:  pb.FileType_File,
				Size:      small_test_file_len,
			},
			save_data:    small_test_file,
			expected_err: nil,
		},
		{
			name: "save in test dir",
			conn_info: &pb.ConnectionRequest{
				Username:  TEST_USER,
				Mode:      pb.ConnectionMode_RDWR,
				Directory: test_dir,
				Filename:  "save_data_test_dir.txt",
				Filetype:  pb.FileType_File,
				Size:      small_test_file_len,
			},
			save_data:    small_test_file,
			expected_err: nil,
		},
		{
			name: "save in uncreated dir",
			conn_info: &pb.ConnectionRequest{
				Username:  TEST_USER,
				Mode:      pb.ConnectionMode_RDWR,
				Directory: "/stay/",
				Filename:  "cool.txt",
				Filetype:  pb.FileType_File,
				Size:      small_test_file_len,
			},
			save_data:    small_test_file,
			expected_err: data.ErrDirNotFound,
		},
		{
			name: "save big file",
			conn_info: &pb.ConnectionRequest{
				Username:  TEST_USER,
				Mode:      pb.ConnectionMode_RDWR,
				Directory: "/",
				Filename:  "save_data_big.txt",
				Filetype:  pb.FileType_File,
				Size:      uint64(len(TEST_FILE_BODY)),
			},
			save_data:    TEST_FILE_BODY,
			expected_err: nil,
		},
		{
			name: "save more than accepted",
			conn_info: &pb.ConnectionRequest{
				Username:  TEST_USER,
				Mode:      pb.ConnectionMode_RDWR,
				Directory: "/",
				Filename:  "save_data_incorrect_chunk.txt",
				Filetype:  pb.FileType_File,
				Size:      small_test_file_len - 5,
			},
			save_data:    small_test_file,
			expected_err: data.ErrUnexpectedFileChange,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			err = saveFile(t.Context(), data_client, test.conn_info, strings.NewReader(test.save_data))

			if test.expected_err != nil {
				if m := getRPCErrorMessage(err); m != test.expected_err.Error() {
					t.Errorf("expected error %v, but got %s", test.expected_err, m)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected nil error, but got %s", getRPCErrorMessage(err))
			}

			// Check file type only
			file, err := os.OpenFile(fmt.Sprintf("%s%s/files%s%s", WORKSPACE_PATH, test.conn_info.Username, test.conn_info.Directory, test.conn_info.Filename), os.O_RDONLY, 0660)
			if err != nil {
				t.Fatal(err)
			}

			got_body_file, err := io.ReadAll(file)
			if err != nil {
				t.Fatal(err)
			}

			if string(got_body_file) != test.save_data {
				t.Error("got file body not implement expected")
			}

		})
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
		MaxChunkSize: 1024,               //byte
		MinChunkSize: 5,                  //byte
		AvailableRAM: 1024 * 1024 * 1024, //byte
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
	file, err := os.OpenFile(fmt.Sprintf("%s%s/files/%s", WORKSPACE_PATH, TEST_USER, test_file_name), os.O_CREATE|os.O_WRONLY, 0660)
	if err != nil {
		t.Fatal(err)
	}

	_, err = file.Write([]byte(TEST_FILE_BODY))
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()

	t.Run("get without connection", func(t *testing.T) {
		random_uuid := uuid.New()
		_, err := data_client.GetData(t.Context(), &pb.GetChunk{
			UUID:    random_uuid.String(),
			ChunkId: 0,
		})

		if m := getRPCErrorMessage(err); m != data.ErrConnectionNotFound.Error() {
			t.Errorf("expected error %s, but got %s", data.ErrConnectionNotFound.Error(), m)
		}
	})

	t.Run("normal get", func(t *testing.T) {
		conn, err := data_client.CreateConnection(t.Context(), &pb.ConnectionRequest{
			Username:  TEST_USER,
			Mode:      pb.ConnectionMode_RDONLY,
			Directory: "/",
			Filename:  test_file_name,
			Filetype:  pb.FileType_File,
		})
		if err != nil {
			t.Fatal(err)
		}

		for i := int32(0); i < conn.ChunksCount; i += 1 {
			t.Run(fmt.Sprintf("get chunk %d", i), func(t *testing.T) {
				ch_id := i

				part, err := data_client.GetData(t.Context(), &pb.GetChunk{
					UUID:    conn.UUID,
					ChunkId: ch_id,
				})
				if err != nil {
					t.Fatal(err)
				}

				offset := uint64(ch_id) * conn.ChunkSize
				expected_chunk := TEST_FILE_BODY[offset : offset+uint64(len(part.Chunk))]
				if string(part.Chunk) != expected_chunk {
					t.Errorf("expected chunk: `%s`, but got `%s`", expected_chunk, string(part.Chunk))
				}
			})
		}
	})
}

func TestGetSum(t *testing.T) {
	if err := createWorkspaceFolders(WORKSPACE_PATH, TEST_USER); err != nil {
		t.Fatal(err)
	}

	max_GRPC_message := 50 * 1024 * 1024

	// Create data grpc client
	grpc_server := grpc.NewServer(grpc.MaxRecvMsgSize(max_GRPC_message), grpc.MaxSendMsgSize(max_GRPC_message))
	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), dataconfig.NewDataServerConfig(WORKSPACE_PATH, dataconfig.DataMemoryConfig{
		MaxChunkSize: uint64(max_GRPC_message) / 2,
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
		data_info      *pb.ConnectionRequest
		gen_file_size  uint64
		bad_sum_wanted bool
	}{
		{
			name: "file 500 bytes",
			data_info: &pb.ConnectionRequest{
				Username:  TEST_USER,
				Mode:      pb.ConnectionMode_RDONLY,
				Directory: "/",
				Filename:  "get_sum_500b.txt",
				Filetype:  pb.FileType_File,
			},
			gen_file_size: 500,
		},
		{
			name: "file 10 kb",
			data_info: &pb.ConnectionRequest{
				Username:  TEST_USER,
				Mode:      pb.ConnectionMode_RDONLY,
				Directory: "/",
				Filename:  "get_sum_10kb.txt",
				Filetype:  pb.FileType_File,
			},
			gen_file_size: 10 * 1024,
		},
		{
			name: "file 500 kb",
			data_info: &pb.ConnectionRequest{
				Username:  TEST_USER,
				Mode:      pb.ConnectionMode_RDONLY,
				Directory: "/",
				Filename:  "get_sum_500kb.txt",
				Filetype:  pb.FileType_File,
			},
			gen_file_size: 500 * 1024,
		},
		{
			name: "file 5 mb",
			data_info: &pb.ConnectionRequest{
				Username:  TEST_USER,
				Mode:      pb.ConnectionMode_RDONLY,
				Directory: "/",
				Filename:  "get_sum_5mb.txt",
				Filetype:  pb.FileType_File,
			},
			gen_file_size: 5 * 1024 * 1024,
		},
		{
			name: "file 50 mb",
			data_info: &pb.ConnectionRequest{
				Username:  TEST_USER,
				Mode:      pb.ConnectionMode_RDONLY,
				Directory: "/",
				Filename:  "get_sum_50mb.txt",
				Filetype:  pb.FileType_File,
			},
			gen_file_size: 50 * 1024 * 1024,
		},
		{
			name: "file 100mb",
			data_info: &pb.ConnectionRequest{
				Username:  TEST_USER,
				Mode:      pb.ConnectionMode_RDONLY,
				Directory: "/",
				Filename:  "get_sum_100mb.txt",
				Filetype:  pb.FileType_File,
			},
			gen_file_size: 100 * 1024 * 1024,
		},
		{
			name: "file 500mb",
			data_info: &pb.ConnectionRequest{
				Username:  TEST_USER,
				Mode:      pb.ConnectionMode_RDONLY,
				Directory: "/",
				Filename:  "get_sum_500mb.txt",
				Filetype:  pb.FileType_File,
			},
			gen_file_size: 500 * 1024 * 1024,
		},
		{
			name: "file 750mb",
			data_info: &pb.ConnectionRequest{
				Username:  TEST_USER,
				Mode:      pb.ConnectionMode_RDONLY,
				Directory: "/",
				Filename:  "get_sum_750mb.txt",
				Filetype:  pb.FileType_File,
			},
			gen_file_size: 750 * 1024 * 1024,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			file_body := genRandomFile(test.gen_file_size)

			// Create test file
			file, err := os.OpenFile(fmt.Sprintf("%s%s/files%s%s", WORKSPACE_PATH, test.data_info.Username, test.data_info.Directory, test.data_info.Filename), os.O_CREATE|os.O_WRONLY, 0660)
			if err != nil {
				t.Fatal(err)
			}

			_, err = file.Write([]byte(file_body))
			if err != nil {
				t.Fatal(err)
			}
			_ = file.Close()

			defer func(test_name string) {
				if test.data_info.Filename == "" {
					return
				}

				if err := os.Remove(fmt.Sprintf("%s%s/files%s%s", WORKSPACE_PATH, test.data_info.Username, test.data_info.Directory, test.data_info.Filename)); err != nil {
					t.Logf("failed remove file; test name = %s; err: %v", test_name, err)
				}
			}(test.name)

			conn, err := data_client.CreateConnection(t.Context(), test.data_info)
			if err != nil {
				t.Fatalf("failed create connection. err: %v", err)
			}

			for i := range conn.ChunksCount {
				got_sum, err := data_client.GetSum(t.Context(), &pb.GetChunk{
					UUID:    conn.UUID,
					ChunkId: i,
				})
				if err != nil {
					t.Fatalf("failed get chunk sum. err: %v", err)
				}

				offset := uint64(i) * conn.ChunkSize
				n := min(offset+conn.ChunkSize, test.gen_file_size)

				expected_sum := sha256.Sum256([]byte(file_body[offset:n]))
				if test.bad_sum_wanted {
					expected_sum[0] = 0
				}

				for j, n := range got_sum.Value {
					if n != expected_sum[j] {
						t.Fatalf("expected sum: %x, but got: %x", string(expected_sum[:]), string(got_sum.Value))
					}
				}
			}
		})
	}
}

// func TestCreateConnection(t *testing.T) {
// 	if err := createWorkspaceFolders(WORKSPACE_PATH, TEST_USER); err != nil {
// 		t.Fatal(err)
// 	}

// 	// Create data grpc client
// 	grpc_server := grpc.NewServer()
// 	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), dataconfig.NewDataServerConfig(WORKSPACE_PATH, dataconfig.DataMemoryConfig{
// 		MaxChunkSize: 25,
// 		MinChunkSize: 5,
// 		AvailableRAM: 1024,
// 	})))

// 	lis, err := net.Listen("tcp", "localhost:8084")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	go func() {
// 		if err := grpc_server.Serve(lis); err != nil {
// 			panic(err)
// 		}
// 	}()

// 	grpc_connection, err := grpc.NewClient("localhost:8084", grpc.WithTransportCredentials(insecure.NewCredentials()))
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	data_client := pb.NewDataServiceClient(grpc_connection)

// 	cases := [...]struct {
// 		name         string
// 		data_info    *pb.DataInfo
// 		expected_err error
// 		conn_check bool
// 	}{
// 		{
// 			name: "empty directory field",
// 			data_info: &pb.DataInfo{
// 				Username:  TEST_USER,
// 				Directory: "",
// 				Filename:  "123.txt",
// 				Filetype:  pb.FileType_File,
// 				Size:      5,
// 			},
// 			expected_err: data.ErrUnspecifiedDir,
// 		},
// 		{
// 			name: "going beyond directory",
// 			data_info: &pb.DataInfo{
// 				Username:  TEST_USER,
// 				Directory: "/../test/",
// 				Filename:  "123.txt",
// 				Filetype:  pb.FileType_File,
// 				Size:      5,
// 			},
// 			expected_err: data.ErrBadDirSyntax,
// 		},
// 		{
// 			name: "directory start is not root",
// 			data_info: &pb.DataInfo{
// 				Username:  TEST_USER,
// 				Directory: "test/test1/",
// 				Filename:  "123.txt",
// 				Filetype:  pb.FileType_File,
// 				Size:      5,
// 			},
// 			expected_err: data.ErrBadDirSyntax,
// 		},
// 		{
// 			name: "empty filename",
// 			data_info: &pb.DataInfo{
// 				Username:  TEST_USER,
// 				Directory: "/",
// 				Filename:  "",
// 				Filetype:  pb.FileType_File,
// 				Size:      5,
// 			},
// 			expected_err: data.ErrEmptyFilename,
// 		},
// 		{
// 			name: "filename bad syntax",
// 			data_info: &pb.DataInfo{
// 				Username:  TEST_USER,
// 				Directory: "/",
// 				Filename:  "123!/xyt$::..txt",
// 				Filetype:  pb.FileType_File,
// 				Size:      5,
// 			},
// 			expected_err: nil, // Todo: add error
// 		},
// 		{
// 			name: "invalid file type",
// 			data_info: &pb.DataInfo{
// 				Username:  TEST_USER,
// 				Directory: "/",
// 				Filename:  "123.txt",
// 				Filetype:  pb.FileType(-10),
// 				Size:      5,
// 			},
// 			expected_err: data.ErrUnexpectedFileType,
// 		},
// 		{
// 			name:         "empty data info",
// 			data_info:    nil,
// 			expected_err: data.ErrInternal,
// 		},
// 		{
// 			name: "read unavailable file connection",
// 			data_info: &pb.DataInfo{
// 				Username: TEST_USER,
// 				Directory: "/",
// 				Filename: "unavailable_file.docx",
// 				Filetype: pb.FileType_File,
// 				Size: 0, // To read
// 			},
// 			expected_err: data.ErrFileNotExist,
// 			conn_check: true,
// 		},
// 		{
// 			name: "normal save connection",
// 			data_info: &pb.DataInfo{
// 				Username:  TEST_USER,
// 				Directory: "/",
// 				Filename:  "test_conn_save.txt",
// 				Filetype:  pb.FileType_File,
// 				Size:      5,
// 			},
// 			conn_check: true,
// 		},
// 		{
// 			name: "normal read connection",
// 			data_info: &pb.DataInfo{
// 				Username: TEST_USER,
// 				Directory: "/",
// 				Filename: "test_conn_read.txt",
// 				Filetype: pb.FileType_File,
// 				Size: 0, // To read
// 			},
// 			conn_check: true,
// 		},
// 		{
// 			name: "rewrite file connection",
// 			data_info: &pb.DataInfo{
// 				Username: TEST_USER,
// 				Directory: "/",
// 				Filename: "test_conn_save.txt",
// 				Filetype: pb.FileType_File,
// 				Size: 5,
// 			},
// 			conn_check: true,
// 		},
// 	}

// 	for _, test := range cases {
// 		t.Run(test.name, func(t *testing.T) {
// 			conn, err :=
// 		})
// 	}
// }
