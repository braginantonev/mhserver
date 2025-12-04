package data

import (
	"context"
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
	return time.Now().Unix() > c.expiration
}

func (c *CachedFile) updateExpiration() {
	c.expiration = time.Now().Add(CACHE_LIFETIME).Unix()
}

func NewCachedFile(file *os.File) *CachedFile {
	return &CachedFile{
		file:       file,
		expiration: time.Now().Add(CACHE_LIFETIME).Unix(),
	}
}

type Cache struct {
	files map[string]*CachedFile
	mux   sync.RWMutex

	clean_time time.Duration
	ctx        context.Context
}

func NewCache(ctx context.Context) *Cache {
	cache := &Cache{
		files:      make(map[string]*CachedFile),
		mux:        sync.RWMutex{},
		clean_time: CACHE_CLEAN_TIME,
		ctx:        ctx,
	}
	go cache.startCleaner()

	return cache
}

func (c *Cache) clean() {
	c.mux.Lock()
	defer c.mux.Unlock()

	for key, file := range c.files {
		if file.isExpired() {
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
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Cache) Push(key string, file *os.File) {
	c.mux.Lock()
	c.files[key] = NewCachedFile(file)
	c.mux.Unlock()
}

// Return true if file exist
func (c *Cache) Get(key string) (*os.File, bool) {
	c.mux.RLock()
	cc_file, ok := c.files[key]
	c.mux.RUnlock()

	if ok {
		cc_file.updateExpiration()
		return cc_file.file, true
	}
	return nil, false
}
