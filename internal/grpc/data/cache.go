package data

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

// todo if needed
type FileMeta struct {
	Size uint64
}

type File struct {
	*os.File
	Meta FileMeta
}

func NewFile(file *os.File, meta FileMeta) File {
	return File{
		File: file,
		Meta: meta,
	}
}

type CachedFile struct {
	file       File
	expiration int64
}

func NewCachedFile(file File) *CachedFile {
	return &CachedFile{
		file:       file,
		expiration: time.Now().Add(FILE_LIFETIME).Unix(),
	}
}

func (p *CachedFile) isExpired() bool {
	return time.Now().Unix() > p.expiration
}

func (p *CachedFile) updateExpiration() {
	p.expiration = time.Now().Add(FILE_LIFETIME).Unix()
}

func (p *CachedFile) GetFile() File {
	return p.file
}

type CachedFiles struct {
	files map[uuid.UUID]*CachedFile
	mux   sync.RWMutex

	ctx           context.Context
	cleanDuration time.Duration
}

func NewCachedFiles(ctx context.Context) *CachedFiles {
	m := &CachedFiles{
		files:         make(map[uuid.UUID]*CachedFile),
		mux:           sync.RWMutex{},
		ctx:           ctx,
		cleanDuration: CLEAN_DURATION,
	}
	go m.startCleaner()

	return m
}

func (m *CachedFiles) clean() {
	m.mux.Lock()
	defer m.mux.Unlock()

	for id, conn := range m.files {
		if conn.isExpired() {
			_ = conn.file.Close()
			delete(m.files, id)
		}
	}
}

func (m *CachedFiles) startCleaner() {
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

func (m *CachedFiles) Push(file CachedFile) uuid.UUID {
	uuid := uuid.New()

	m.mux.Lock()
	m.files[uuid] = &file
	m.mux.Unlock()

	return uuid
}

func (m *CachedFiles) Get(uuid uuid.UUID) (*File, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	info, ok := m.files[uuid]
	if !ok {
		return nil, false
	}

	info.updateExpiration()
	return &info.file, true
}

// Return count active files UUIDs. If count is 0, return 1 by default
func (m *CachedFiles) Length() int {
	m.mux.RLock()
	map_ln := len(m.files)
	m.mux.RUnlock()

	// Standard value
	if map_ln == 0 {
		return 1
	}

	return map_ln
}
