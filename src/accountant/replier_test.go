package accountant

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestAppendSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repl, err := newReplierBuffer(ctx, time.Second)
	assert.NilError(t, err)

	for i := 0; i < maxArraySize; i++ {
		token := make([]byte, 32)
		rand.Read(token)
		err = repl.insert(newMemory(&Vertex{Hash: [32]byte(token)}))
		assert.NilError(t, err)
	}
}

func TestAppendOverflowMaxArrSize(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repl, err := newReplierBuffer(ctx, time.Second)
	assert.NilError(t, err)

	for i := 0; i < maxArraySize; i++ {
		token := make([]byte, 32)
		rand.Read(token)
		err = repl.insert(newMemory(&Vertex{Hash: [32]byte(token)}))
		assert.NilError(t, err)
	}

	err = repl.insert(newMemory(&Vertex{}))
	assert.Error(t, err, ErrNotEnoughSpace.Error())
}

func TestAppendSubscribeCorrectOrder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Millisecond * 102)
		cancel()
	}()

	vrxNum := 100
	repl, err := newReplierBuffer(ctx, time.Millisecond)
	assert.NilError(t, err)

	go func() {
		var counter int
		var m memory
		for v := range repl.subscribe() {
			if v.vrx == nil {
				break
			}
			counter++
			if m.vrx == nil {
				m = v
				continue
			}

			if m.vrx.CreatedAt.Before(v.vrx.CreatedAt) {
				err = fmt.Errorf(
					"new vertex crated at [ %v ] is before last vertex created at [ %v ]",
					v.vrx.CreatedAt, m.vrx.CreatedAt,
				)
				assert.NilError(t, err)
			}

			m = v
		}

		assert.Equal(t, counter, vrxNum)
	}()

	vertexes := make([]*Vertex, 0, vrxNum)

	for i := 0; i < vrxNum; i++ {
		token := make([]byte, 32)
		rand.Read(token)
		vertexes = append(vertexes, &Vertex{
			Hash:      [32]byte(token),
			CreatedAt: time.Now(),
		})
	}
	for i, j := 0, len(vertexes)-1; i < j; i, j = i+1, j-1 {
		vertexes[i], vertexes[j] = vertexes[j], vertexes[i]
	}

	for i := range vertexes {
		err = repl.insert(newMemory(vertexes[i]))
		assert.NilError(t, err)
	}
	<-ctx.Done()
}
