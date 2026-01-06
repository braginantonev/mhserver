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

	TEST_FILE_BODY string = `Антон Чигур никого не убивал!
Антон Чигур никого не покарал!
Антон Чигур ничего не уничтожал!`
)

// Create server workspace in to test files with `File` type only
func createWorkspaceFolders(workspace_path, username string) error {
	return os.MkdirAll(fmt.Sprintf("%s%s/files", workspace_path, username), 0700)
}

// Client simulation
func saveFile(ctx context.Context, data_client pb.DataServiceClient, data_info *pb.DataInfo, reader io.Reader) error {
	conn, err := data_client.CreateConnection(ctx, data_info)
	if err != nil {
		return err
	}

	log.Printf("save file (%s) with size %d; chunk size = %d, count = %d", data_info.Filename, data_info.Size, conn.ChunkSize, conn.ChunksCount)

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
		func(ch_id int, data []byte) {
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
	test_file_name := "save_data_test_file.txt"

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

		if m := getRPCErrorMessage(err); m != data.ErrUnexpectedFileChange.Error() {
			t.Errorf("expected error %s, but got %s", data.ErrUnexpectedFileChange.Error(), m)
		}
	})

	t.Run("incorrect chunk size", func(t *testing.T) {
		test_data_info := &pb.DataInfo{
			Username: TEST_USER,
			Filename: "incorrect size.txt",
			Filetype: pb.FileType_File,
			Size:     uint64(len(TEST_FILE_BODY)),
		}

		conn, err := data_client.CreateConnection(t.Context(), test_data_info)
		if err != nil {
			t.Fatal(err)
		}

		_, err = data_client.SaveData(t.Context(), &pb.SaveChunk{
			UUID: conn.UUID,
			Data: &pb.FilePart{
				Chunk: []byte(TEST_FILE_BODY + "garbage"),
			},
		})

		if m := getRPCErrorMessage(err); m != data.ErrIncorrectChunkSize.Error() {
			t.Errorf("expected error %s, but got %s", data.ErrIncorrectChunkSize.Error(), m)
		}
	})

	t.Run("normal save", func(t *testing.T) {
		data_info := &pb.DataInfo{
			Username: TEST_USER,
			Filename: test_file_name,
			Filetype: pb.FileType_File,
			Size:     uint64(len(TEST_FILE_BODY)),
		}

		err = saveFile(t.Context(), data_client, data_info, strings.NewReader(TEST_FILE_BODY))
		if err != nil {
			t.Fatal(err)
		}

		// Check file type only
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
	})
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
	file, err := os.OpenFile(fmt.Sprintf("%s%s/files/%s", WORKSPACE_PATH, TEST_USER, test_file_name), os.O_CREATE|os.O_RDWR, 0660)
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

		if m := getRPCErrorMessage(err); m != data.ErrUnexpectedFileChange.Error() {
			t.Errorf("expected error %s, but got %s", data.ErrUnexpectedFileChange.Error(), m)
		}
	})

	// Test
	test_file_info := &pb.DataInfo{
		Username: TEST_USER,
		Filename: test_file_name,
		Filetype: pb.FileType_File,
		Size:     uint64(len(TEST_FILE_BODY)),
	}

	t.Run("normal get", func(t *testing.T) {
		conn, err := data_client.CreateConnection(t.Context(), test_file_info)
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
		data_info      *pb.DataInfo
		bad_sum_wanted bool
	}{
		{
			name: "file 500 bytes",
			data_info: &pb.DataInfo{
				Username: TEST_USER,
				Filename: "500b.txt",
				Filetype: pb.FileType_File,
				Size:     500,
			},
		},
		{
			name: "file 10 kb",
			data_info: &pb.DataInfo{
				Username: TEST_USER,
				Filename: "10kb.txt",
				Filetype: pb.FileType_File,
				Size:     10 * 1024,
			},
		},
		{
			name: "file 500 kb",
			data_info: &pb.DataInfo{
				Username: TEST_USER,
				Filename: "500kb.txt",
				Filetype: pb.FileType_File,
				Size:     500 * 1024,
			},
		},
		{
			name: "file 5 mb",
			data_info: &pb.DataInfo{
				Username: TEST_USER,
				Filename: "5mb.txt",
				Filetype: pb.FileType_File,
				Size:     5 * 1024 * 1024,
			},
		},
		{
			name: "file 50 mb",
			data_info: &pb.DataInfo{
				Username: TEST_USER,
				Filename: "50mb.txt",
				Filetype: pb.FileType_File,
				Size:     50 * 1024 * 1024,
			},
		},
		{
			name: "file 100mb",
			data_info: &pb.DataInfo{
				Username: TEST_USER,
				Filename: "100mb.txt",
				Filetype: pb.FileType_File,
				Size:     100 * 1024 * 1024,
			},
		},
		{
			name: "file 500mb",
			data_info: &pb.DataInfo{
				Username: TEST_USER,
				Filename: "500mb.txt",
				Filetype: pb.FileType_File,
				Size:     500 * 1024 * 1024,
			},
		},
		{
			name: "file 750mb",
			data_info: &pb.DataInfo{
				Username: TEST_USER,
				Filename: "750mb.txt",
				Filetype: pb.FileType_File,
				Size:     750 * 1024 * 1024,
			},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			file_body := genRandomFile(test.data_info.Size)

			if err := saveFile(t.Context(), data_client, test.data_info, strings.NewReader(file_body)); err != nil {
				t.Fatal(err)
			}

			defer func(test_name string) {
				if test.data_info.Filename == "" {
					return
				}

				if err := os.Remove(fmt.Sprintf("%s%s/files/%s", WORKSPACE_PATH, test.data_info.Username, test.data_info.Filename)); err != nil {
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
				n := min(offset+conn.ChunkSize, test.data_info.Size)

				expected_sum := sha256.Sum256([]byte(file_body[offset:n]))
				if test.bad_sum_wanted {
					expected_sum[0] = 0
				}

				for j, n := range got_sum.Sum {
					if n != expected_sum[j] {
						t.Fatalf("expected sum: %x, but got: %x", string(expected_sum[:]), string(got_sum.Sum))
					}
				}
			}
		})
	}
}
