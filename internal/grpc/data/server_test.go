package data_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/braginantonev/mhserver/internal/config"
	"github.com/braginantonev/mhserver/internal/grpc/data"
	"github.com/braginantonev/mhserver/internal/repository/dirs"
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

// streaming save
func saveFile(ctx context.Context, data_client pb.DataServiceClient, req_file *pb.RequiredFile, reader io.Reader) error {
	file_id, err := data_client.InitFile(ctx, req_file)
	if err != nil {
		return err
	}

	stream, err := data_client.SaveFile(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = stream.CloseSend() }()

sendLoop:
	for i := uint64(0); ; i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
			chunk := make([]byte, file_id.MaxChunkSize)
			n, err := reader.Read(chunk)
			if err != nil && err != io.EOF {
				return err
			}

			if n == 0 && err == io.EOF {
				break sendLoop
			}

			if err := stream.Send(&pb.SaveFileChunk{
				Id: file_id.FileID,
				Value: &pb.Chunk{
					Data:   chunk[:n],
					Offset: file_id.MaxChunkSize * i,
				},
			}); err != nil {
				return err
			}
		}
	}

	_, err = stream.CloseAndRecv()
	return err
}

func errorIs(err error, target error) bool {
	// Standard check
	if errors.Is(err, target) {
		return true
	}

	// GRPC error check

	target_desc := ""
	if target != nil {
		target_desc = target.Error()
	}

	st, ok := status.FromError(err)
	if ok {
		return st.Message() == target_desc
	}

	return false
}

func TestInitFile(t *testing.T) {
	if err := createWorkspaceFolders(WORKSPACE_PATH, TEST_USER); err != nil {
		t.Fatal(err)
	}

	// Create data grpc client
	grpc_server := grpc.NewServer()
	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), data.NewDataServerConfig(WORKSPACE_PATH, config.MemoryConfig{
		MaxChunkSize: 25,
		MinChunkSize: 5,
		Allocated:    1024 * 1024 * 1024,
	})))

	lis, err := net.Listen("tcp", "localhost:8084")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := grpc_server.Serve(lis); err != nil {
			panic(err)
		}
	}()

	grpc_connection, err := grpc.NewClient("localhost:8084", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}

	data_client := pb.NewDataServiceClient(grpc_connection)

	cases := [...]struct {
		name         string
		req_file     *pb.RequiredFile
		expected_err error
	}{
		{
			name: "empty directory field",
			req_file: &pb.RequiredFile{
				Dir: &pb.Directory{
					User:  TEST_USER,
					Value: "",
				},
				Name:    "123.txt",
				NewSize: new(uint64(5)),
			},
			expected_err: dirs.ErrBadDirSyntax,
		},
		{
			name: "going beyond directory",
			req_file: &pb.RequiredFile{
				Dir: &pb.Directory{
					User:  TEST_USER,
					Value: "/../test/",
				},
				Name:    "123.txt",
				NewSize: new(uint64(5)),
			},
			expected_err: dirs.ErrBadDirSyntax,
		},
		{
			name: "directory start is not root",
			req_file: &pb.RequiredFile{
				Dir: &pb.Directory{
					User:  TEST_USER,
					Value: "test/test1/",
				},
				Name:    "123.txt",
				NewSize: new(uint64(5)),
			},
			expected_err: dirs.ErrBadDirSyntax,
		},
		{
			name: "empty filename",
			req_file: &pb.RequiredFile{
				Dir: &pb.Directory{
					User:  TEST_USER,
					Value: "/",
				},
				Name:    "",
				NewSize: new(uint64(5)),
			},
			expected_err: data.ErrBadFilenameSyntax,
		},
		{
			name: "filename bad syntax",
			req_file: &pb.RequiredFile{
				Dir: &pb.Directory{
					User:  TEST_USER,
					Value: "/",
				},
				Name:    "123/.txt",
				NewSize: new(uint64(5)),
			},
			expected_err: data.ErrBadFilenameSyntax,
		},
		{
			name: "init from uncreated directory request",
			req_file: &pb.RequiredFile{
				Dir: &pb.Directory{
					User:  TEST_USER,
					Value: "/uncreated_dir/",
				},
				Name:    "123.txt",
				NewSize: new(uint64(5)),
			},
			expected_err: data.ErrDirNotFound,
		},
		{
			name: "normal init",
			req_file: &pb.RequiredFile{
				Dir: &pb.Directory{
					User:  TEST_USER,
					Value: "/",
				},
				Name:    "test_normal_init.txt",
				NewSize: new(uint64(5)),
			},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := data_client.InitFile(t.Context(), test.req_file)

			if !errorIs(err, test.expected_err) {
				t.Errorf("expected %v but got %v", test.expected_err, err)
			}
		})
	}

	t.Run("size change", func(t *testing.T) {
		t.Parallel()

		const DEFAULT_FILE_BODY string = "12345"

		req_file := &pb.RequiredFile{
			Dir: &pb.Directory{
				User:  TEST_USER,
				Value: "/",
			},
			Name:    "test_truncate.txt",
			NewSize: nil, // will be set later in test
		}

		f, err := os.Create(fmt.Sprintf("%s%s/%s", WORKSPACE_PATH, req_file.Dir.User, req_file.Name))
		if err != nil {
			t.Fatalf("failed create test file: %s", err)
		}

		if _, err = f.WriteString(DEFAULT_FILE_BODY); err != nil {
			t.Fatalf("failed write test file: %s", err)
		}

		test_size := func(t *testing.T) {
			_, err = data_client.InitFile(t.Context(), req_file)
			if err != nil {
				t.Errorf("failed init file: %s", err)
			}

			if info, err := f.Stat(); err != nil {
				t.Errorf("failed get file stat: %s", err)
			} else {
				if info.Size() != int64(len(DEFAULT_FILE_BODY)) {
					t.Errorf("expected file size: %d, but got: %d", len(DEFAULT_FILE_BODY), info.Size())
				}
			}
		}

		t.Run("without new size", func(t *testing.T) {
			test_size(t)
		})

		t.Run("with new size", func(t *testing.T) {
			new_size := new(uint64(len(DEFAULT_FILE_BODY)) + 15)
			req_file.NewSize = new_size

			test_size(t)

			*new_size = uint64(len(DEFAULT_FILE_BODY)) / 2

			test_size(t)
		})
	})
}

func TestSaveFile(t *testing.T) {
	if err := createWorkspaceFolders(WORKSPACE_PATH, TEST_USER); err != nil {
		t.Fatal(err)
	}

	max_chunk_size := 10

	grpc_server := grpc.NewServer(grpc.MaxRecvMsgSize(max_chunk_size + 256))
	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), data.NewDataServerConfig(WORKSPACE_PATH, config.MemoryConfig{
		MaxChunkSize: uint64(max_chunk_size), //byte
		MinChunkSize: 5,                      //byte
		Allocated:    1024 * 1024 * 1024,     //byte
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

	t.Run("without init file", func(t *testing.T) {
		random_uuid := uuid.New()
		stream, err := data_client.SaveFile(t.Context())
		if err != nil {
			t.Fatalf("failed create stream: %s", err)
		}

		if err = stream.Send(&pb.SaveFileChunk{
			Id: &pb.FileID{Value: random_uuid.String()},
			Value: &pb.Chunk{
				Data: []byte("be be be"),
			},
		}); err != nil {
			t.Fatalf("failed send chunk: %s", err)
		}

		if _, err = stream.CloseAndRecv(); !errorIs(err, data.ErrConnectionNotFound) {
			t.Errorf("expected error %v, but got %v", data.ErrConnectionNotFound, err)
		}
	})

	// To test: "save in test dir"
	test_dir := "/save_data_test_dir/"
	if err = os.MkdirAll(fmt.Sprintf("%s%s/files%s", WORKSPACE_PATH, TEST_USER, test_dir), 0770); err != nil {
		t.Fatal(err)
	}

	small_test_file := "I use arch btw"
	small_test_file_len := uint64(len(small_test_file))

	cases := [...]struct {
		name         string
		req_file     *pb.RequiredFile
		save_data    string
		expected_err error
	}{
		{
			name: "save in root dir",
			req_file: &pb.RequiredFile{
				Dir: &pb.Directory{
					User:  TEST_USER,
					Value: "/",
				},
				Name:    "save_data_single.txt",
				NewSize: new(small_test_file_len),
			},
			save_data:    small_test_file,
			expected_err: nil,
		},
		{
			name: "save in test dir",
			req_file: &pb.RequiredFile{
				Dir: &pb.Directory{
					User:  TEST_USER,
					Value: test_dir,
				},
				Name:    "test.txt",
				NewSize: new(small_test_file_len),
			},
			save_data:    small_test_file,
			expected_err: nil,
		},
		{
			name: "save in uncreated dir",
			req_file: &pb.RequiredFile{
				Dir: &pb.Directory{
					User:  TEST_USER,
					Value: "/uncreated_dir/",
				},
				Name:    "cool.txt",
				NewSize: new(small_test_file_len),
			},
			save_data:    small_test_file,
			expected_err: data.ErrDirNotFound,
		},
		{
			name: "save big file",
			req_file: &pb.RequiredFile{
				Dir: &pb.Directory{
					User:  TEST_USER,
					Value: "/",
				},
				Name:    "save_data_big.txt",
				NewSize: new(uint64(len(TEST_FILE_BODY))),
			},
			save_data:    TEST_FILE_BODY,
			expected_err: nil,
		},
		{
			name: "save in out of file",
			req_file: &pb.RequiredFile{
				Dir: &pb.Directory{
					User:  TEST_USER,
					Value: "/",
				},
				Name:    "save_data_incorrect_chunk.txt",
				NewSize: new(small_test_file_len - 5),
			},
			save_data:    small_test_file,
			expected_err: data.ErrUnexpectedFileChange,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			err = saveFile(t.Context(), data_client, test.req_file, strings.NewReader(test.save_data))

			if test.expected_err != nil {
				if !errorIs(err, test.expected_err) {
					t.Errorf("expected error %v, but got %v", test.expected_err, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected nil error, but got %v", err)
			}

			// Check file type only
			file, err := os.OpenFile(fmt.Sprintf("%s%s/files%s%s", WORKSPACE_PATH, test.req_file.Dir.User, test.req_file.Dir.Value, test.req_file.Name), os.O_RDONLY, 0660)
			if err != nil {
				t.Fatal(err)
			}

			got_body_file, err := io.ReadAll(file)
			if err != nil {
				t.Fatal(err)
			}

			if string(got_body_file) != test.save_data {
				t.Error("got file body not implement than expected")
			}

		})
	}
}

/*
func TestGetData(t *testing.T) {
	test_file_name := "get_data_test_file.txt"

	if err := createWorkspaceFolders(WORKSPACE_PATH, TEST_USER); err != nil {
		t.Fatal(err)
	}

	// Create data grpc client
	grpc_server := grpc.NewServer()
	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), data.NewDataServerConfig(WORKSPACE_PATH, config.MemoryConfig{
		MaxChunkSize: 1024,               //byte
		MinChunkSize: 5,                  //byte
		Allocated:    1024 * 1024 * 1024, //byte
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

		if !errorIs(err, data.ErrConnectionNotFound) {
			t.Errorf("expected error %v, but got %v", data.ErrConnectionNotFound, err)
		}
	})

	t.Run("normal get", func(t *testing.T) {
		conn, err := data_client.CreateConnection(t.Context(), &pb.ConnectionRequest{
			Username:  TEST_USER,
			Mode:      pb.ConnectionMode_RDONLY,
			Directory: "/",
			Filename:  test_file_name,
		})
		if err != nil {
			t.Fatal(err)
		}

		for i := uint32(0); i < conn.ChunksCount; i += 1 {
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
	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), data.NewDataServerConfig(WORKSPACE_PATH, config.MemoryConfig{
		MaxChunkSize: uint64(max_GRPC_message) / 2,
		MinChunkSize: 4 * 1024,
		Allocated:    1024 * 1024 * 1024,
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

	// Вместо создания всей строки в памяти
	genRandomFile := func(size uint64) (*os.File, error) {
		file, err := os.CreateTemp(WORKSPACE_PATH, fmt.Sprintf("%d-*.txt", size))
		if err != nil {
			return nil, err
		}

		const CHUNK_SIZE = 64 * 1024
		buffer := make([]byte, CHUNK_SIZE)
		letters := []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ\n\t")

		for written := uint64(0); written < size; {
			toWrite := CHUNK_SIZE
			if size-written < CHUNK_SIZE {
				toWrite = int(size - written)
			}

			for i := 0; i < toWrite; i++ {
				buffer[i] = letters[rand.Intn(len(letters))]
			}

			n, err := file.Write(buffer[:toWrite])
			if err != nil {
				return nil, err
			}
			written += uint64(n)
		}

		return file, nil
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
			},
			gen_file_size: 750 * 1024 * 1024,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			expected, err := genRandomFile(test.gen_file_size)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				_ = os.Remove(expected.Name())
			}()

			// Create test file
			filepath := fmt.Sprintf("%s%s/files%s%s", WORKSPACE_PATH, test.data_info.Username, test.data_info.Directory, test.data_info.Filename)
			file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY, 0660)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				_ = os.Remove(filepath)
			}()

			_, err = io.Copy(file, expected)
			if err != nil {
				t.Fatal(err)
			}
			_ = file.Close()

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

				expected_data := make([]byte, conn.ChunkSize)
				read, err := expected.ReadAt(expected_data, int64(uint64(i)*conn.ChunkSize))
				if err != nil {
					t.Fatal(err)
				}

				expected_sum := sha256.Sum256(expected_data[:read])
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

func TestGetFiles(t *testing.T) {
	if err := createWorkspaceFolders(WORKSPACE_PATH, TEST_USER); err != nil {
		t.Fatal(err)
	}

	test_dir := "/get_files_test/"
	if err := os.MkdirAll(WORKSPACE_PATH+TEST_USER+"/files"+test_dir, 0770); err != nil {
		t.Fatal(err)
	}

	// Create data grpc client
	grpc_server := grpc.NewServer()
	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), data.NewDataServerConfig(WORKSPACE_PATH, config.MemoryConfig{
		MaxChunkSize: 1024,               //byte
		MinChunkSize: 5,                  //byte
		Allocated:    1024 * 1024 * 1024, //byte
	})))

	lis, err := net.Listen("tcp", "localhost:8085")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := grpc_server.Serve(lis); err != nil {
			panic(err)
		}
	}()

	grpc_connection, err := grpc.NewClient("localhost:8085", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}

	data_client := pb.NewDataServiceClient(grpc_connection)

	extensions := []string{"jpg", "png", "txt", "doc", "docx", "1c", "svg"}
	gen_filename := func(with_ext bool) string {
		filename := uuid.New().String()
		if with_ext {
			filename += "." + extensions[rand.Intn(len(extensions))]
		}
		return filename
	}

	// ! Кейсы должны выполняться строго последовательно
	cases := [...]struct {
		name          string
		target_dir    string
		files_count   int // Files which will be created in target dir
		folders_count int // Dirs which will be created in target dir
		expected_err  error
	}{
		{
			name:         "empty dir request",
			target_dir:   "",
			expected_err: dirs.ErrBadDirSyntax,
		},
		{
			name:         "bad dir syntax",
			target_dir:   "/../",
			expected_err: dirs.ErrBadDirSyntax,
		},
		{
			name:         "empty directory",
			target_dir:   test_dir,
			expected_err: nil,
		},
		{
			name:          "with one dir",
			target_dir:    test_dir,
			folders_count: 1,
			expected_err:  nil,
		},
		{
			name:         "with one file",
			target_dir:   test_dir,
			files_count:  1,
			expected_err: nil,
		},
		{
			name:         "with 50 files",
			target_dir:   test_dir,
			files_count:  50,
			expected_err: nil,
		},
		{
			name:          "with 50 folders",
			target_dir:    test_dir,
			folders_count: 50,
			expected_err:  nil,
		},
		{
			name:          "100 files and dirs",
			target_dir:    test_dir,
			files_count:   100,
			folders_count: 100,
			expected_err:  nil,
		},
	}

	for _, test := range cases {
		workspace_dir := WORKSPACE_PATH + TEST_USER + "/files" + test.target_dir

		// Create folders
		for range test.folders_count {
			if err := os.Mkdir(workspace_dir+gen_filename(false), 0660); err != nil {
				t.Fatalf("failed create folders: %v", err)
			}
		}

		// Create files
		for range test.files_count {
			if _, err := os.Create(workspace_dir + gen_filename(true)); err != nil {
				t.Fatalf("failed create files: %v", err)
			}
		}

		expected_files, err := os.ReadDir(workspace_dir)
		if err != nil {
			t.Fatalf("failed read test dir: %v", err)
		}

		t.Run(test.name, func(t *testing.T) {
			files_list, err := data_client.GetFiles(t.Context(), &pb.Directory{
				User:  TEST_USER,
				Value: test.target_dir,
			})

			if !errorIs(err, test.expected_err) {
				t.Fatalf("expected error: %v, but got: %v", test.expected_err, err)
			}

			if test.files_count == 0 && test.folders_count == 0 {
				return
			}

			for i, file := range files_list.Value {
				expected_info, err := expected_files[i].Info()
				if err != nil {
					t.Fatalf("failed get file info: %v", err)
				}

				expected_file_info := pb.FileInfo{
					Name:    expected_files[i].Name(),
					IsDir:   expected_files[i].IsDir(),
					Size:    uint64(expected_info.Size()),
					ModTime: uint64(expected_info.ModTime().Unix()),
				}

				if file.Name != expected_file_info.Name {
					t.Errorf("expected filename: %s, but got: %s", expected_file_info.Name, file.Name)
				}

				if file.IsDir != expected_file_info.IsDir {
					t.Errorf("expected isDir: %t, but got: %t", expected_file_info.IsDir, file.IsDir)
				}

				if file.Size != expected_file_info.Size {
					t.Errorf("expected file size: %d, but got: %d", expected_file_info.Size, file.Size)
				}

				if file.ModTime != expected_file_info.ModTime {
					t.Errorf("expected modTime: %d, but got: %d", expected_file_info.ModTime, file.ModTime)
				}
			}
		})

		for _, file := range expected_files {
			if test.target_dir != test_dir {
				continue
			}

			if err = os.RemoveAll(workspace_dir + file.Name()); err != nil {
				t.Fatalf("failed cleanup created test files: %v", err)
			}
		}
	}

	// Test get files with not empty dir
	if err := os.MkdirAll(WORKSPACE_PATH+TEST_USER+"/files"+test_dir+"test1/test2/test3", 0770); err != nil {
		t.Fatalf("failed create test dirs: %v", err)
	}

	t.Run("with dir contained another dir", func(t *testing.T) {
		files, err := data_client.GetFiles(t.Context(), &pb.Directory{
			User:  TEST_USER,
			Value: test_dir,
		})

		if err != nil {
			t.Fatalf("expected nil error, but got: %v", err)
		}

		if len(files.Value) != 1 {
			t.Errorf("expected one file in dir, but got: %d", len(files.Value))
		}

		if files.Value[0].Name != "test1" {
			t.Errorf("expected dir name: test1, but got: %s", files.Value[0].Name)
		}
	})

	t.Run("dir not found", func(t *testing.T) {
		_, err := data_client.GetFiles(t.Context(), &pb.Directory{
			User:  TEST_USER,
			Value: "/unexpected_dir/",
		})

		if !errorIs(err, data.ErrDirNotFound) {
			t.Fatalf("expected ErrDirNotFound error, but got: %v", err)
		}
	})

	if err := os.RemoveAll(WORKSPACE_PATH + TEST_USER + "/files" + test_dir); err != nil {
		t.Errorf("failed cleanup: %v", err)
	}
}
*/
