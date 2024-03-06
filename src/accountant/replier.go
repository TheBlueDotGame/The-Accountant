package accountant

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
)

const (
	maxArraySize = 500
	maxRepeats   = 25
)

const longevity = time.Minute

var (
	ErrNotEnoughSpace           = errors.New("not enough space in the replier buffer")
	ErrVertexRepetitionExceeded = errors.New("vertex repetition exceeded")
)

type memory struct {
	vrx      *Vertex
	repeated int
}

func newMemory(v *Vertex) memory {
	return memory{vrx: v, repeated: 0}
}

func bytesToHex(b []byte) string {
	return fmt.Sprintf("%x", b)
}

// buffer stores Vertex' and then replay Vertex after set time tick.
// It stores the Vertex sorted end replies them in proper order based on created at time.
type buffer struct {
	pub     chan memory
	members []memory
}

func (b *buffer) getNext() memory {
	if len(b.members) == 0 {
		return memory{}
	}

	slices.SortStableFunc(b.members, func(a, b memory) int {
		if a.vrx == nil || b.vrx == nil {
			return 0
		}
		if a.vrx.CreatedAt.Before(a.vrx.CreatedAt) {
			return -1
		}
		if a.vrx.CreatedAt.After(a.vrx.CreatedAt) {
			return 1
		}
		return 0
	})

	m := b.members[0]
	b.members = b.members[1:]

	return m
}

func (b *buffer) run(ctx context.Context, ts time.Duration) {
	tc := time.NewTicker(ts)
	defer tc.Stop()
	defer close(b.pub)

ticker:
	for {
		select {
		case <-tc.C:
			v := b.getNext()
			if v.vrx == nil {
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
		pub:     make(chan memory, maxArraySize),
		members: make([]memory, 0, maxArraySize),
	}

	go buf.run(ctx, tick)

	return buf, nil
}

func (b buffer) subscribe() <-chan memory {
	return b.pub
}

func (b *buffer) insert(m memory) error {
	if len(b.members) == maxArraySize {
		return ErrNotEnoughSpace
	}

	if m.repeated > maxRepeats {
		return ErrVertexRepetitionExceeded
	}

	m.repeated++

	b.members = append(b.members, m)
	return nil
}
