package fileuuidmap

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	FILE_UUID_LIFETIME time.Duration = 1 * time.Minute
	CLEAN_DURATION     time.Duration = 5 * time.Second
)

type FilePath struct {
	path       string
	expiration int64
}

func NewFilePath(path string) FilePath {
	return FilePath{
		path:       path,
		expiration: time.Now().Add(FILE_UUID_LIFETIME).Unix(),
	}
}

func (p FilePath) isExpired() bool {
	return time.Now().Unix() > p.expiration
}

func (p *FilePath) updateExpiration() {
	p.expiration = time.Now().Add(FILE_UUID_LIFETIME).Unix()
}

type FileUUIDMap struct {
	paths map[uuid.UUID]FilePath
	mux   *sync.RWMutex

	ctx            context.Context
	clean_duration time.Duration
}

func NewFileUUIDMap(ctx context.Context) *FileUUIDMap {
	m := &FileUUIDMap{
		paths:          make(map[uuid.UUID]FilePath),
		mux:            &sync.RWMutex{},
		ctx:            ctx,
		clean_duration: CLEAN_DURATION,
	}
	go m.startCleaner()

	return m
}

func (m *FileUUIDMap) clean() {
	m.mux.Lock()
	defer m.mux.Unlock()

	for key, path := range m.paths {
		if path.isExpired() {
			delete(m.paths, key)
		}
	}
}

func (m *FileUUIDMap) startCleaner() {
	ticker := time.NewTicker(m.clean_duration)
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

func (m *FileUUIDMap) Add(path string) uuid.UUID {
	uuid := uuid.New()

	m.mux.Lock()
	m.paths[uuid] = NewFilePath(path)
	m.mux.Unlock()

	return uuid
}

func (m *FileUUIDMap) Get(uuid uuid.UUID) (string, bool) {
	m.mux.RLock()
	got, ok := m.paths[uuid]
	m.mux.RUnlock()

	if !ok {
		return "", false
	}

	got.updateExpiration()
	return got.path, true
}
