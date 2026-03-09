package fileuuidmap

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	FILE_LIFETIME  time.Duration = 2 * time.Minute
	CLEAN_DURATION time.Duration = 10 * time.Second
)

type ChunksInfo struct {
	ChunkSize uint64
	Count     int
	Loaded    int
}

func NewChunksInfo(c_size uint64, c_count int) ChunksInfo {
	return ChunksInfo{
		ChunkSize: c_size,
		Count:     c_count,
	}
}

type File struct {
	*os.File

	path       string
	chunks     ChunksInfo
	expiration int64
}

func NewFile(file *os.File, path string, chunks_info ChunksInfo) *File {
	return &File{
		File:       file,
		path:       path,
		chunks:     chunks_info,
		expiration: time.Now().Add(FILE_LIFETIME).Unix(),
	}
}

func (p *File) isExpired() bool {
	return time.Now().Unix() > p.expiration
}

func (p *File) updateExpiration() {
	p.expiration = time.Now().Add(FILE_LIFETIME).Unix()
}

func (p *File) GetPath() string {
	return p.path
}

func (p *File) GetChunkSize() uint64 {
	return p.chunks.ChunkSize
}

func (p *File) GetChunksCount() int {
	return p.chunks.Count
}

func (p *File) GetLoadedChunks() int {
	return p.chunks.Loaded
}

type FileUUIDMap struct {
	files map[uuid.UUID]*File
	mux   *sync.RWMutex

	ctx           context.Context
	cleanDuration time.Duration
}

func NewFileUUIDMap(ctx context.Context) *FileUUIDMap {
	m := &FileUUIDMap{
		files:         make(map[uuid.UUID]*File),
		mux:           &sync.RWMutex{},
		ctx:           ctx,
		cleanDuration: CLEAN_DURATION,
	}
	go m.startCleaner()

	return m
}

func (m *FileUUIDMap) clean() {
	m.mux.Lock()
	defer m.mux.Unlock()

	for id, file := range m.files {
		if file.isExpired() {
			file.Close()
			delete(m.files, id)
		}
	}
}

func (m *FileUUIDMap) startCleaner() {
	ticker := time.NewTicker(m.cleanDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.clean()
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *FileUUIDMap) Push(file *File) uuid.UUID {
	uuid := uuid.New()

	m.mux.Lock()
	m.files[uuid] = file
	m.mux.Unlock()

	return uuid
}

func (m *FileUUIDMap) Get(uuid uuid.UUID) (*File, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	info, ok := m.files[uuid]
	if !ok {
		return nil, false
	}

	info.updateExpiration()
	return info, true
}

// Update loaded chunks counter from file. Return ErrFileNotFound if file by uuid is not found.
// Return EOC if loaded chunks >= his count. This is end for all connections.
func (m *FileUUIDMap) UpdateLoadedChunks(uuid uuid.UUID) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	info, ok := m.files[uuid]
	if !ok {
		return ErrFileNotFound
	}

	info.chunks.Loaded += 1
	info.updateExpiration()

	if info.chunks.Loaded == info.chunks.Count {
		delete(m.files, uuid)
	}

	return nil
}

// Return count active files UUIDs. If count is 0, return 1 by default
func (m *FileUUIDMap) Length() int {
	m.mux.RLock()
	map_ln := len(m.files)
	m.mux.RUnlock()

	// Standard value
	if map_ln == 0 {
		return 1
	}

	return map_ln
}

// Size of disk space, which will be saved. Calculate size unsaved chunks.
func (m *FileUUIDMap) ExpectedSavedSpace() uint64 {
	m.mux.RLock()
	defer m.mux.RUnlock()

	var res uint64
	for _, info := range m.files {
		res += info.GetChunkSize() * uint64(info.GetChunksCount()-info.GetLoadedChunks())
	}
	return res
}
