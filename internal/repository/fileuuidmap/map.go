package fileuuidmap

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	FILE_INFO_UUID_LIFETIME time.Duration = 5 * time.Minute
	CLEAN_DURATION          time.Duration = 10 * time.Second
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

type FileInfo struct {
	path   string
	chunks ChunksInfo

	expiration int64
}

func NewFileInfo(path string, chunks_info ChunksInfo) *FileInfo {
	return &FileInfo{
		path:       path,
		chunks:     chunks_info,
		expiration: time.Now().Add(FILE_INFO_UUID_LIFETIME).Unix(),
	}
}

func (p *FileInfo) isExpired() bool {
	return time.Now().Unix() > p.expiration
}

func (p *FileInfo) updateExpiration() {
	p.expiration = time.Now().Add(FILE_INFO_UUID_LIFETIME).Unix()
}

func (p *FileInfo) GetPath() string {
	return p.path
}

func (p *FileInfo) GetChunkSize() uint64 {
	return p.chunks.ChunkSize
}

func (p *FileInfo) GetChunksCount() int {
	return p.chunks.Count
}

func (p *FileInfo) GetLoadedChunks() int {
	return p.chunks.Loaded
}

type FileUUIDMap struct {
	infos map[uuid.UUID]*FileInfo
	mux   *sync.RWMutex

	ctx           context.Context
	cleanDuration time.Duration
}

func NewFileUUIDMap(ctx context.Context) *FileUUIDMap {
	m := &FileUUIDMap{
		infos:         make(map[uuid.UUID]*FileInfo),
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

	for key, path := range m.infos {
		if path.isExpired() {
			delete(m.infos, key)
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

func (m *FileUUIDMap) Add(path string, chunks_info ChunksInfo) uuid.UUID {
	uuid := uuid.New()

	m.mux.Lock()
	m.infos[uuid] = NewFileInfo(path, chunks_info)
	m.mux.Unlock()

	return uuid
}

func (m *FileUUIDMap) Get(uuid uuid.UUID) (*FileInfo, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	info, ok := m.infos[uuid]
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

	info, ok := m.infos[uuid]
	if !ok {
		return ErrFileNotFound
	}

	if info.chunks.Count >= info.chunks.Loaded {
		delete(m.infos, uuid)
		return EOC
	}

	info.chunks.Loaded += 1
	info.updateExpiration()

	return nil
}

// Return count active files UUIDs. If count is 0, return 1 by default
func (m *FileUUIDMap) Length() int {
	m.mux.RLock()
	map_ln := len(m.infos)
	m.mux.RUnlock()

	// Standard value
	if map_ln == 0 {
		return 1
	}

	return map_ln
}
