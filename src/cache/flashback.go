package cache

import (
	"errors"
	"time"

	"github.com/allegro/bigcache"
)

const (
	lifeWindow       = time.Second * 20
	cleanWindow      = time.Second * 10
	hardMaxCacheSize = 5
	haxEntrySize     = 0
)

var ErrWrongHashSizeInFlash = errors.New("wrong error size supplied to flashback has hash")

type Flashback struct {
	mem *bigcache.BigCache
}

// NewFlash creates the new Hippocampus on success or returns an error otherwise.
func NewFlash() (*Flashback, error) {
	c, err := bigcache.NewBigCache(bigcache.Config{
		Shards:           shards,
		LifeWindow:       lifeWindow,
		CleanWindow:      cleanWindow,
		HardMaxCacheSize: hardMaxCacheSize,
		MaxEntrySize:     haxEntrySize,
	})
	if err != nil {
		return nil, err
	}

	return &Flashback{mem: c}, nil
}

// HasHash checks if flashback received given hash in last ten seconds.
func (f *Flashback) HasHash(h []byte) (bool, error) {
	if len(h) != 32 {
		return false, ErrWrongHashSizeInFlash
	}

	defer f.mem.Set(string(h), []byte{})

	_, err := f.mem.Get(string(h))
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, bigcache.ErrEntryNotFound) {
		return false, err
	}
	return false, nil
}

// HasAddress checks if flashback received given address request.
func (f *Flashback) HasAddress(a string) (bool, error) {
	defer f.mem.Set(a, []byte{})

	_, err := f.mem.Get(a)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, bigcache.ErrEntryNotFound) {
		return false, err
	}

	return false, nil
}

// RemoveAddress removes address from the cache.
func (f *Flashback) RemoveAddress(a string) error {
	return f.mem.Delete(a)
}

// Close closes the cache in a safe way allowing all the goroutines to finish their jobs and cleaning the heap.
func (f *Flashback) Close() error {
	err := f.mem.Close()
	if errors.Is(err, bigcache.ErrEntryNotFound) {
		return nil
	}
	return nil
}
