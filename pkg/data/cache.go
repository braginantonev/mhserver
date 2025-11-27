package data

import (
	"os"
	"sync"
	"time"
)

const (
	CACHE_LIFETIME   time.Duration = 15 * time.Second
	CACHE_CLEAN_TIME time.Duration = 3 * time.Second
)

type CachedFile struct {
	file       *os.File
	expiration int64
}

func (c *CachedFile) close() {
	_ = c.file.Close()
}

func (c *CachedFile) isExpired() bool {
	return time.Now().Unix() < c.expiration
}

func NewCachedFile(file *os.File, expiration int64) *CachedFile {
	return &CachedFile{
		file:       file,
		expiration: expiration,
	}
}

type Cache struct {
	files map[string]*CachedFile
	mux   sync.RWMutex

	clean_time time.Duration
	close_ch   chan struct{}
}

func NewCache() *Cache {
	cache := &Cache{
		files:      make(map[string]*CachedFile),
		mux:        sync.RWMutex{},
		clean_time: CACHE_CLEAN_TIME,
		close_ch:   make(chan struct{}),
	}
	cache.startCleaner()

	return cache
}

func (c *Cache) clean() {
	c.mux.Lock()
	defer c.mux.Unlock()

	for key, file := range c.files {
		if !file.isExpired() {
			file.close()
			delete(c.files, key)
		}
	}
}

func (c *Cache) startCleaner() {
	ticker := time.NewTicker(c.clean_time)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.clean()
		case <-c.close_ch:
			return
		}
	}
}

func (c *Cache) Push(key string, file *os.File) {
	c.mux.Lock()
	c.files[key] = NewCachedFile(file, time.Now().Unix())
	c.mux.Unlock()
}

// Return true if file exist
func (c *Cache) Get(key string) (*os.File, bool) {
	c.mux.RLock()
	cc_file, ok := c.files[key]
	c.mux.RUnlock()

	if ok {
		return cc_file.file, true
	}

	return nil, false
}

func (c *Cache) Close() {
	close(c.close_ch)
}
