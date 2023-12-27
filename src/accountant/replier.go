package accountant

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"
)

const (
	maxArraySize = 200
	maxRepeats   = 4
)

const longevity = time.Minute

var (
	ErrNotEnoughSpace           = errors.New("not enough space in the replier buffer")
	ErrVertexRepetitionExceeded = errors.New("vertex repetition exceeded")
)

type memory struct {
	repeted   int
	createdAt time.Time
}

func bytesToHex(b []byte) string {
	return fmt.Sprintf("%x", b)
}

// buffer stores Vertex' and then replay Vertex after set time tick.
// It stores the Vertex sorted end replies them in proper order based on created at time.
type buffer struct {
	pub       chan *Vertex
	members   []*Vertex
	accounter map[string]memory
}

func (b *buffer) cleanup() {
	t := time.Now()
	maps.DeleteFunc(b.accounter, func(k string, m memory) bool {
		return m.createdAt.Add(longevity).Before(t)
	})
}

func (b *buffer) getNext() *Vertex {
	if len(b.members) == 0 {
		return nil
	}

	slices.SortStableFunc(b.members, func(a, b *Vertex) int {
		if a == nil || b == nil {
			return 0
		}
		if a.CreatedAt.Before(a.CreatedAt) {
			return -1
		}
		if a.CreatedAt.After(a.CreatedAt) {
			return 1
		}
		return 0
	})

	vrx := b.members[0]
	b.members = b.members[1:]

	return vrx
}

func (b *buffer) run(ctx context.Context, ts time.Duration) {
	tc := time.NewTicker(ts)
	defer tc.Stop()
	defer close(b.pub)

ticker:
	for {
		select {
		case <-tc.C:
			b.cleanup()
			v := b.getNext()
			if v == nil {
				continue ticker
			}
			b.pub <- v
		case <-ctx.Done():
			break ticker
		}
	}
}

func newReplierBuffer(ctx context.Context, tick time.Duration) (*buffer, error) {
	buf := &buffer{
		pub:       make(chan *Vertex, maxArraySize),
		members:   make([]*Vertex, 0, maxArraySize),
		accounter: make(map[string]memory),
	}

	go buf.run(ctx, tick)

	return buf, nil
}

func (b buffer) subscribe() <-chan *Vertex {
	return b.pub
}

func (b *buffer) insert(v *Vertex) error {
	if len(b.members) == maxArraySize {
		return ErrNotEnoughSpace
	}

	h := bytesToHex(v.Hash[:])
	m, ok := b.accounter[h]
	if ok && m.repeted > 5 {
		return ErrVertexRepetitionExceeded
	}
	if !ok {
		m.createdAt = time.Now()
	}
	m.repeted++
	b.accounter[h] = m

	b.members = append(b.members, v)
	return nil
}
