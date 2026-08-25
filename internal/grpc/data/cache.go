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

func NewCachedFile(file File) CachedFile {
	return CachedFile{
		file:       file,
		expiration: time.Now().Add(FILE_LIFETIME).Unix(),
	}
}

func (p CachedFile) isExpired() bool {
	return time.Now().Unix() > p.expiration
}

func (p CachedFile) GetFile() File {
	return p.file
}

type CachedFiles struct {
	files map[uuid.UUID]CachedFile
	mux   sync.RWMutex

	ctx           context.Context
	cleanDuration time.Duration
}

func NewCachedFiles(ctx context.Context) *CachedFiles {
	m := &CachedFiles{
		files:         make(map[uuid.UUID]CachedFile),
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

func (m *CachedFiles) Push(file File) uuid.UUID {
	uuid := uuid.New()

	m.mux.Lock()
	m.files[uuid] = NewCachedFile(file)
	m.mux.Unlock()

	return uuid
}

func (m *CachedFiles) Get(uuid uuid.UUID) (File, bool) {
	m.mux.Lock()
	defer m.mux.Unlock()

	info, ok := m.files[uuid]
	if !ok {
		return File{}, false
	}

	delete(m.files, uuid)
	return info.file, true
}
