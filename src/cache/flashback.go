package cache

import (
	"errors"
	"time"

	"github.com/allegro/bigcache"
)

var ErrWrongHashSizeInFlash = errors.New("wrong error size supplied to flashback has hash")

type Flashback struct {
	mem *bigcache.BigCache
}

func NewFlash() (*Flashback, error) {
	c, err := bigcache.NewBigCache(bigcache.Config{
		Shards:           shards,
		LifeWindow:       time.Second * 10,
		CleanWindow:      time.Second * 5,
		HardMaxCacheSize: 5,
		MaxEntrySize:     0,
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

	_, err := f.mem.Get(string(h))
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, bigcache.ErrEntryNotFound) {
		return false, err
	}
	err = f.mem.Set(string(h), []byte{})
	if err != nil {
		return false, err
	}
	return false, nil
}

// Close closes the cache in a safe way allowing all the goroutines to finish their jobs and cleaning the heap.
func (f *Flashback) Close() error {
	return f.mem.Close()
}
