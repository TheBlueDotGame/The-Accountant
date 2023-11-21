package dataprovider

import (
	"bytes"
	"context"
	"crypto/rand"
	"sync"
	"time"
)

// Config holds configuration for Cache.
type Config struct {
	Longevity uint64 `yaml:"longevity"` // Data longevity in seconds.
}

type data struct {
	raw       []byte
	timestamp int64
}

// Cache is a simple in-memory cache for storing generated data.
type Cache struct {
	data      map[string]data
	mux       sync.RWMutex
	longevity time.Duration
}

// New creates new Cache and runs the cleaner.
func New(ctx context.Context, cfg Config) *Cache {
	if cfg.Longevity == 0 {
		cfg.Longevity = 60
	}
	longevity := time.Duration(cfg.Longevity) * time.Second
	c := &Cache{
		data:      make(map[string]data),
		mux:       sync.RWMutex{},
		longevity: longevity,
	}
	go func(ctx context.Context, t time.Duration, c *Cache) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(t * 2):
				c.clean()
			}
		}
	}(ctx, longevity, c)

	return c
}

func (c *Cache) clean() {
	c.mux.Lock()
	defer c.mux.Unlock()
	now := time.Now().UnixNano()
	for k, v := range c.data {
		if v.timestamp < now {
			delete(c.data, k)
		}
	}
}

// ProvideData generates data and stores it referring to given address.
func (c *Cache) ProvideData(address string) []byte {
	c.mux.Lock()
	defer c.mux.Unlock()

	buf := make([]byte, 128)
	rand.Read(buf)
	c.data[address] = data{
		raw:       buf,
		timestamp: time.Now().Add(c.longevity).UnixNano(),
	}

	return buf
}

// ValidateData checks if data is stored for given address and is not expired.
func (c *Cache) ValidateData(address string, data []byte) bool {
	c.mux.RLock()
	defer c.mux.RUnlock()

	d, ok := c.data[address]
	if !ok {
		return false
	}

	if d.timestamp < time.Now().UnixNano() {
		return false
	}

	return bytes.Equal(data, d.raw)
}
