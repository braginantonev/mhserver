package data_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braginantonev/mhserver/internal/interceptors"
	"github.com/braginantonev/mhserver/internal/services/data"
	"github.com/braginantonev/mhserver/pkg/contextkeys"
	pb "github.com/braginantonev/mhserver/proto/gen/data"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const (
	WORKSPACE_PATH  string = "/tmp/mhserver_tests/"
	TEST_USER       string = "user"
	CHUNK_SIZE      int    = 1024
	SAVE_CHUNK_SIZE int    = 5

	TEST_FILE_BODY string = `- Скажи, дружище, ты стихи любишь?
	- Стихи? Ну, не особо, сэр.
	- Тихо в лесу, только не спит только медведь... Он ещё с вечера начал пердеть. Вот и не спит медведь.
	...
	- Тихо в лесу, только не спит ёж. Нюхает ёж медвежий пердёжь, вот и не спит ёж.
	- Эм... Что?
	- Тихо в лесу, только не спит сова. Есть у совы смешная трава, вот и не спит сова.
	- Сэр, может, заправку закончим?`
)

// Create server workspace in to test files with `File` type only
func createWorkspaceFolders(workspace_path, username string, user_spaces []string) error {
	for _, usp := range user_spaces {
		if err := os.MkdirAll(filepath.Join(workspace_path, usp, username), 0777); err != nil {
			return err
		}
	}
	return nil
}

// streaming save
func saveFile(ctx context.Context, data_client pb.DataServiceClient, req_file *pb.RequiredFile, reader io.Reader) error {
	stream, err := data_client.SaveFile(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = stream.CloseSend() }()

	if err = stream.Send(&pb.SaveFileChunk{
		Info: &pb.SaveFileChunk_Meta{
			Meta: req_file,
		},
	}); err != nil {
		return err
	}

sendLoop:
	for i := uint64(0); ; i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
			chunk := make([]byte, SAVE_CHUNK_SIZE)
			n, err := reader.Read(chunk)
			if err != nil && err != io.EOF {
				return err
			}

			if n == 0 && err == io.EOF {
				break sendLoop
			}

			if err := stream.Send(&pb.SaveFileChunk{
				Info: &pb.SaveFileChunk_Chunk{
					Chunk: &pb.Chunk{
						Data:   chunk[:n],
						Offset: uint64(SAVE_CHUNK_SIZE) * i,
					},
				},
			}); err != nil {
				return err
			}
		}
	}

	_, err = stream.CloseAndRecv()
	return err
}

func TestSaveFile(t *testing.T) {
	if err := createWorkspaceFolders(WORKSPACE_PATH, TEST_USER, []string{"0"}); err != nil {
		t.Fatal(err)
	}

	auth_intc := interceptors.NewFakeAuthInterceptor(TEST_USER)

	grpc_server := grpc.NewServer(
		grpc.UnaryInterceptor(auth_intc.Unary),
		grpc.StreamInterceptor(auth_intc.Stream),
	)

	data_config := data.NewDataServerConfig(WORKSPACE_PATH, []string{"0"})
	data_config.ChunkSize = uint64(SAVE_CHUNK_SIZE)

	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), data_config))

	lis := bufconn.Listen(SAVE_CHUNK_SIZE)
	go grpc_server.Serve(lis)

	grpc_connection, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}

	data_client := pb.NewDataServiceClient(grpc_connection)

	// To test: "save in test dir"
	test_dir := "/save_data_test_dir/"
	if err = os.MkdirAll(filepath.Join(WORKSPACE_PATH, "0", TEST_USER, test_dir), 0777); err != nil {
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
				if !errors.Is(err, test.expected_err) {
					t.Errorf("expected error %v, but got %v", test.expected_err, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected nil error, but got %v", err)
			}

			// Check file type only
			file, err := os.OpenFile(filepath.Join(WORKSPACE_PATH, "0", TEST_USER, test.req_file.Dir.Value, test.req_file.Name), os.O_RDONLY, 0777)
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

func TestReadFile(t *testing.T) {
	test_file_name := "get_data_test_file.txt"

	if err := createWorkspaceFolders(WORKSPACE_PATH, TEST_USER, []string{"0"}); err != nil {
		t.Fatal(err)
	}

	auth_intc := interceptors.NewFakeAuthInterceptor(TEST_USER)

	grpc_server := grpc.NewServer(
		grpc.UnaryInterceptor(auth_intc.Unary),
		grpc.StreamInterceptor(auth_intc.Stream),
	)

	data_config := data.NewDataServerConfig(WORKSPACE_PATH, []string{"0"})
	data_config.ChunkSize = uint64(CHUNK_SIZE)

	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(t.Context(), data_config))

	lis := bufconn.Listen(1024 * 1024)
	go grpc_server.Serve(lis)

	grpc_connection, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}

	data_client := pb.NewDataServiceClient(grpc_connection)

	// Create test file
	file, err := os.OpenFile(filepath.Join(WORKSPACE_PATH, "0", TEST_USER, test_file_name), os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		t.Fatal(err)
	}

	_, err = file.Write([]byte(TEST_FILE_BODY))
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()

	t.Run("normal get", func(t *testing.T) {
		req := &pb.RequiredFile{
			Dir: &pb.Directory{
				Value: "/",
			},
			Name: test_file_name,
		}

		stream, err := data_client.ReadFile(t.Context(), req)
		if err != nil {
			t.Fatalf("failed create connection (%v)", err)
		}

		for {
			v, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Fatalf("failed get chunk (%v)", err)
			}

			expected := TEST_FILE_BODY[v.Offset : v.Offset+uint64(len(v.Data))]
			if string(v.Data) != expected {
				t.Errorf("expected chunk: `%s`, but got: `%s`", expected, string(v.Data))
			}
		}
	})
}

func TestGetSum(t *testing.T) {
	if err := createWorkspaceFolders(WORKSPACE_PATH, TEST_USER, []string{"0"}); err != nil {
		t.Fatal(err)
	}

	data_config := data.NewDataServerConfig(WORKSPACE_PATH, []string{"0"})
	data_config.ChunkSize = uint64(CHUNK_SIZE)

	data_service := data.NewDataServer(t.Context(), data_config)

	// Вместо создания всей строки в памяти
	genRandomFile := func(size uint64) (*os.File, error) {
		file, err := os.CreateTemp(filepath.Join(WORKSPACE_PATH, "0", TEST_USER), fmt.Sprintf("%d-*.txt", size))
		if err != nil {
			return nil, err
		}

		buffer := make([]byte, CHUNK_SIZE)
		letters := []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ\n\t")

		for written := uint64(0); written < size; {
			toWrite := CHUNK_SIZE
			if size-written < uint64(CHUNK_SIZE) {
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

		if err = file.Sync(); err != nil {
			return nil, err
		}

		_, err = file.Seek(0, 0)
		if err != nil {
			return nil, err
		}

		return file, nil
	}

	cases := [...]struct {
		name           string
		req_file       *pb.RequiredFile
		gen_file_size  uint64
		bad_sum_wanted bool
	}{
		{
			name: "file 500 bytes",
			req_file: &pb.RequiredFile{
				Dir: &pb.Directory{
					Value: "/",
				},
				// we don't provide a filename because we set him later after file gen
			},
			gen_file_size: 500,
		},
		{
			name: "file 10 kb",
			req_file: &pb.RequiredFile{
				Dir: &pb.Directory{
					Value: "/",
				},
			},
			gen_file_size: 10 * 1024,
		},
		{
			name: "file 500 kb",
			req_file: &pb.RequiredFile{
				Dir: &pb.Directory{
					Value: "/",
				},
			},
			gen_file_size: 500 * 1024,
		},
		{
			name: "file 5 mb",
			req_file: &pb.RequiredFile{
				Dir: &pb.Directory{
					Value: "/",
				},
			},
			gen_file_size: 5 * 1024 * 1024,
		},
		{
			name: "file 50 mb",
			req_file: &pb.RequiredFile{
				Dir: &pb.Directory{
					Value: "/",
				},
			},
			gen_file_size: 50 * 1024 * 1024,
		},
		{
			name: "file 100mb",
			req_file: &pb.RequiredFile{
				Dir: &pb.Directory{
					Value: "/",
				},
			},
			gen_file_size: 100 * 1024 * 1024,
		},
		{
			name: "file 500mb",
			req_file: &pb.RequiredFile{
				Dir: &pb.Directory{
					Value: "/",
				},
			},
			gen_file_size: 500 * 1024 * 1024,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			test_file, err := genRandomFile(test.gen_file_size)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				_ = os.Remove(test_file.Name())
			}()

			expected_sum := sha256.New()

			_, err = io.Copy(expected_sum, test_file)
			if err != nil {
				t.Fatal(err)
			}
			_ = test_file.Close()

			test.req_file.Name = test_file.Name()[strings.LastIndexByte(test_file.Name(), '/')+1:]

			req_ctx := context.WithValue(t.Context(), contextkeys.USERNAME, TEST_USER)
			got, err := data_service.GetSum(req_ctx, test.req_file)
			if err != nil {
				t.Fatalf("failed get sum (%v)", err)
			}

			if string(expected_sum.Sum(nil)) != string(got.Value) {
				t.Errorf("expected sum: `%s`, but got: %s", string(expected_sum.Sum(nil)), string(got.Value))
			}
		})
	}
}

func TestGetFiles(t *testing.T) {
	if err := createWorkspaceFolders(WORKSPACE_PATH, TEST_USER, []string{"0"}); err != nil {
		t.Fatal(err)
	}

	test_dir := "/get_files_test/"
	if err := os.MkdirAll(filepath.Join(WORKSPACE_PATH, "0", TEST_USER, test_dir), 0777); err != nil {
		t.Fatal(err)
	}

	data_config := data.NewDataServerConfig(WORKSPACE_PATH, []string{"0"})
	data_config.ChunkSize = uint64(CHUNK_SIZE)

	data_service := data.NewDataServer(t.Context(), data_config)

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
			expected_err: data.ErrBadDirSyntax,
		},
		{
			name:         "bad dir syntax",
			target_dir:   "/../",
			expected_err: data.ErrBadDirSyntax,
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
		workspace_dir := filepath.Join(WORKSPACE_PATH, "0", TEST_USER, test.target_dir)

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
			req_ctx := context.WithValue(t.Context(), contextkeys.USERNAME, TEST_USER)
			files_list, err := data_service.GetFiles(req_ctx, &pb.Directory{
				Value: test.target_dir,
			})

			if !errors.Is(err, test.expected_err) {
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
	if err := os.MkdirAll(filepath.Join(WORKSPACE_PATH, "0", TEST_USER, test_dir, "test1/test2/test3"), 0777); err != nil {
		t.Fatalf("failed create test dirs: %v", err)
	}

	t.Run("with dir contained another dir", func(t *testing.T) {
		req_ctx := context.WithValue(t.Context(), contextkeys.USERNAME, TEST_USER)
		files, err := data_service.GetFiles(req_ctx, &pb.Directory{
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
		req_ctx := context.WithValue(t.Context(), contextkeys.USERNAME, TEST_USER)
		_, err := data_service.GetFiles(req_ctx, &pb.Directory{
			Value: "/unexpected_dir/",
		})

		if !errors.Is(err, data.ErrDirNotFound) {
			t.Fatalf("expected ErrDirNotFound error, but got: %v", err)
		}
	})

	if err := os.RemoveAll(WORKSPACE_PATH + TEST_USER + "/files" + test_dir); err != nil {
		t.Errorf("failed cleanup: %v", err)
	}
}
