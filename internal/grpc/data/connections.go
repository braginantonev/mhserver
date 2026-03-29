package data

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	pb "github.com/braginantonev/mhserver/proto/data"
	"github.com/google/uuid"
)

const (
	FILE_LIFETIME  time.Duration = 2 * time.Minute
	CLEAN_DURATION time.Duration = 10 * time.Second
)

var (
	ErrFileNotFound = errors.New("file not found. Bad uuid")
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

	path   string
	chunks ChunksInfo
}

func (f File) GetPath() string {
	return f.path
}

func (f File) GetChunksInfo() ChunksInfo {
	return f.chunks
}

func (f File) IsLoaded() bool {
	return f.chunks.Loaded >= f.chunks.Count
}

func NewFile(file *os.File, path string, chunks_info ChunksInfo) File {
	return File{
		File:   file,
		path:   path,
		chunks: chunks_info,
	}
}

type Connection struct {
	mode       pb.ConnectionMode
	file       File
	expiration int64
}

func NewConnection(file File, mode pb.ConnectionMode) *Connection {
	return &Connection{
		file:       file,
		mode:       mode,
		expiration: time.Now().Add(FILE_LIFETIME).Unix(),
	}
}

func (p *Connection) isExpired() bool {
	return time.Now().Unix() > p.expiration
}

func (p *Connection) updateExpiration() {
	p.expiration = time.Now().Add(FILE_LIFETIME).Unix()
}

func (p *Connection) GetFile() File {
	return p.file
}

type Connections struct {
	value map[uuid.UUID]*Connection
	mux   *sync.RWMutex

	ctx           context.Context
	cleanDuration time.Duration
}

func NewConnectionsMap(ctx context.Context) *Connections {
	m := &Connections{
		value:         make(map[uuid.UUID]*Connection),
		mux:           &sync.RWMutex{},
		ctx:           ctx,
		cleanDuration: CLEAN_DURATION,
	}
	go m.startCleaner()

	return m
}

func (m *Connections) clean() {
	m.mux.Lock()
	defer m.mux.Unlock()

	for id, conn := range m.value {
		if conn.isExpired() {
			_ = conn.file.Close()
			delete(m.value, id)
		}
	}
}

func (m *Connections) startCleaner() {
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

func (m *Connections) Push(file *Connection) uuid.UUID {
	uuid := uuid.New()

	m.mux.Lock()
	m.value[uuid] = file
	m.mux.Unlock()

	return uuid
}

func (m *Connections) Get(uuid uuid.UUID) (*Connection, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	info, ok := m.value[uuid]
	if !ok {
		return nil, false
	}

	info.updateExpiration()
	return info, true
}

// Update loaded chunks counter from file. Return ErrFileNotFound if file by uuid is not found.
func (m *Connections) UpdateLoadedFileChunks(uuid uuid.UUID) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	info, ok := m.value[uuid]
	if !ok {
		return ErrFileNotFound
	}

	info.file.chunks.Loaded += 1
	info.updateExpiration()

	return nil
}

// Return count active files UUIDs. If count is 0, return 1 by default
func (m *Connections) Length() int {
	m.mux.RLock()
	map_ln := len(m.value)
	m.mux.RUnlock()

	// Standard value
	if map_ln == 0 {
		return 1
	}

	return map_ln
}

// Size of disk space, which will be saved. Calculate size unsaved chunks.
func (m *Connections) ExpectedSavedSpace() uint64 {
	m.mux.RLock()
	defer m.mux.RUnlock()

	var res uint64
	for _, conn := range m.value {
		res += conn.file.chunks.ChunkSize * uint64(conn.file.chunks.Count-conn.file.chunks.Loaded)
	}
	return res
}
