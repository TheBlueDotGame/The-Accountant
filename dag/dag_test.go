package dag

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"github.com/bartossh/Computantis/transaction"
	msgpackv2 "github.com/shamaton/msgpack/v2"
	"github.com/vmihailenco/msgpack"
	"gotest.tools/v3/assert"
)

func generateData(l int) []byte {
	data := make([]byte, 0, l)
	for i := 0; i < l; i++ {
		data = append(data, byte(rand.Intn(255)))
	}
	return data
}

func BenchmarkSerializartion(b *testing.B) {
	v := Vertex{
		Transaction: transaction.Transaction{
			Hash:              [32]byte(generateData(32)),
			CreatedAt:         time.Now(),
			Subject:           "Test packing to binary",
			Data:              generateData(2048),
			IssuerAddress:     string(generateData(64)),
			ReceiverAddress:   string(generateData(64)),
			IssuerSignature:   generateData(32),
			ReceiverSignature: generateData(32),
		},
		Hash:            [32]byte(generateData(32)),
		LeftParentHash:  [32]byte(generateData(32)),
		RightParentHash: [32]byte(generateData(32)),
	}

	b.Run("json marshal", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			json.Marshal(v)
		}
	})

	b.Run("gob marshal", func(b *testing.B) {
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			enc.Encode(v)
		}
	})

	b.Run("msgpack marshal", func(b *testing.B) {
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			msgpack.Marshal(v)
		}
	})

	b.Run("msgpack v2 marshal", func(b *testing.B) {
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			msgpackv2.Marshal(v)
		}
	})

	b.Run("json unmarshal", func(b *testing.B) {
		buf, _ := json.Marshal(v)
		var newV Vertex
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			json.Unmarshal(buf, &newV)
		}
	})

	b.Run("gob unmarshal", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			b.StopTimer()
			buf := bytes.NewBuffer(make([]byte, 0))
			enc := gob.NewEncoder(buf)
			enc.Encode(v)
			var newV Vertex
			dec := gob.NewDecoder(buf)
			b.StartTimer()
			err := dec.Decode(&newV)
			assert.NilError(b, err)
		}
	})

	b.Run("msgpack unmarshal", func(b *testing.B) {
		buf, _ := msgpack.Marshal(v)
		b.ResetTimer()
		var newV Vertex
		for n := 0; n < b.N; n++ {
			msgpack.Unmarshal(buf, &newV)
		}
	})

	b.Run("msgpack v2 unmarshal", func(b *testing.B) {
		buf, _ := msgpackv2.Marshal(v)
		b.ResetTimer()
		var newV Vertex
		for n := 0; n < b.N; n++ {
			msgpackv2.Unmarshal(buf, &newV)
		}
	})
}

func TestCorrectness(t *testing.T) {
	v := Vertex{
		Transaction: transaction.Transaction{
			Hash:              [32]byte(generateData(32)),
			CreatedAt:         time.Now(),
			Subject:           "Test packing to binary",
			Data:              generateData(2048),
			IssuerAddress:     string(generateData(64)),
			ReceiverAddress:   string(generateData(64)),
			IssuerSignature:   generateData(32),
			ReceiverSignature: generateData(32),
		},
		Hash:            [32]byte(generateData(32)),
		LeftParentHash:  [32]byte(generateData(32)),
		RightParentHash: [32]byte(generateData(32)),
	}

	t.Run("msgpack", func(t *testing.T) {
		buf, _ := msgpack.Marshal(v)
		var newV Vertex
		err := msgpack.Unmarshal(buf, &newV)
		assert.NilError(t, err)
		assert.DeepEqual(t, v, newV)
	})

	t.Run("msgpack v2", func(t *testing.T) {
		buf, _ := msgpackv2.Marshal(v)
		var newV Vertex
		err := msgpackv2.Unmarshal(buf, &newV)
		assert.NilError(t, err)
		assert.DeepEqual(t, v, newV)
	})

	t.Run("msgpack (marshal) msgpack v2 (unmarshal)", func(t *testing.T) {
		buf, _ := msgpack.Marshal(v)
		var newV Vertex
		err := msgpackv2.Unmarshal(buf, &newV)
		assert.NilError(t, err)
		assert.DeepEqual(t, v, newV)
	})
}
