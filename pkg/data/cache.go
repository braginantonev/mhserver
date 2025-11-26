package data

import (
	"os"
	"sync"
	"time"
)

const (
	CACHE_LIFETIME time.Duration = 30 * time.Second
)

type CachedFile struct {
	file *os.File

	lifetime      time.Duration
	creation_time time.Time
}

func (c *CachedFile) Close() {
	_ = c.file.Close()
}

func (c *CachedFile) IsAlive() bool {
	return time.Since(c.creation_time) < c.lifetime
}

func NewCachedFile(file *os.File) *CachedFile {
	return &CachedFile{
		file:          file,
		lifetime:      CACHE_LIFETIME,
		creation_time: time.Now(),
	}
}

type Cache struct {
	files map[string]*CachedFile
	mux   sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{
		files: make(map[string]*CachedFile),
		mux:   sync.RWMutex{},
	}
}

func (c *Cache) clean() {
	c.mux.Lock()
	defer c.mux.Unlock()

	for key, file := range c.files {
		if !file.IsAlive() {
			file.Close()
			delete(c.files, key)
		}
	}
}

func (c *Cache) Push(key string, file *os.File) {
	c.mux.RLock()
	c.files[key] = NewCachedFile(file)
	c.mux.RUnlock()
}

// Return true if file exist
func (c *Cache) Get(key string) (*os.File, bool) {
	c.mux.RLock()
	file, ok := c.files[key]
	c.mux.RUnlock()

	if ok && file.IsAlive() {
		return file.file, true
	} else {
		c.clean()
		return nil, false
	}
}
