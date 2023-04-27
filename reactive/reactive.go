package reactive

import "sync"

type subscriber[T any] struct {
	c         chan T
	container *Observable[T]
}

// Cancel removes observable from container.
// To cancel observable, call this method.
// Not calling this method may result in memory leak.
func (o *subscriber[T]) Cancel() {
	o.container.delete(o)
	close(o.c)
}

// Channel returns channel that can be used to read from observable.
func (o *subscriber[T]) Channel() <-chan T {
	return o.c
}

// Observable creates a container for subscribers.
// This works in single producer multiple consumer pattern.
type Observable[T any] struct {
	mux         sync.RWMutex
	subscribers map[*subscriber[T]]struct{}
	size        int
}

// New creates Observable container that holds channels for all subscribers.
// size is the buffer size of each channel.
func New[T any](size int) *Observable[T] {
	return &Observable[T]{
		mux:         sync.RWMutex{},
		subscribers: make(map[*subscriber[T]]struct{}),
		size:        size,
	}
}

// Subscribe subscribes to the container.
func (o *Observable[T]) Subscribe() *subscriber[T] {
	obs := &subscriber[T]{
		c:         make(chan T, o.size),
		container: o,
	}
	o.mux.Lock()
	defer o.mux.Unlock()
	o.subscribers[obs] = struct{}{}
	return obs
}

// Publish publishes value to all subscribers.
func (o *Observable[T]) Publish(v T) {
	o.mux.RLock()
	defer o.mux.RUnlock()
	for c := range o.subscribers {
		c.c <- v
	}
}

func (o *Observable[T]) delete(c *subscriber[T]) {
	o.mux.Lock()
	defer o.mux.Unlock()
	delete(o.subscribers, c)
}
