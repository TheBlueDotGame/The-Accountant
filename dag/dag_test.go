package dag

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/bartossh/Computantis/logging"
	"github.com/bartossh/Computantis/spice"
	"github.com/bartossh/Computantis/stdoutwriter"
	"github.com/bartossh/Computantis/transaction"
	"github.com/bartossh/Computantis/wallet"
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
		CreatedAt: time.Now(),
		Transaction: transaction.Transaction{
			Hash:              [32]byte(generateData(32)),
			CreatedAt:         time.Now(),
			Subject:           "Test packing to binary",
			Data:              generateData(2048),
			IssuerAddress:     string(generateData(64)),
			ReceiverAddress:   string(generateData(64)),
			IssuerSignature:   generateData(32),
			ReceiverSignature: generateData(32),
			Spice:             spice.New(math.MaxUint64, 100),
		},
		Hash:            [32]byte(generateData(32)),
		LeftParentHash:  [32]byte(generateData(32)),
		RightParentHash: [32]byte(generateData(32)),
		Weight:          100,
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
		CreatedAt: time.Now(),
		Transaction: transaction.Transaction{
			Hash:              [32]byte(generateData(32)),
			CreatedAt:         time.Now(),
			Subject:           "Test packing to binary",
			Data:              generateData(2048),
			IssuerAddress:     string(generateData(64)),
			ReceiverAddress:   string(generateData(64)),
			IssuerSignature:   generateData(32),
			ReceiverSignature: generateData(32),
			Spice:             spice.New(math.MaxUint64, 100),
		},
		Hash:            [32]byte(generateData(32)),
		LeftParentHash:  [32]byte(generateData(32)),
		RightParentHash: [32]byte(generateData(32)),
		Weight:          100,
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

func TestDagStart(t *testing.T) {
	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Faield with error: %s\n", err)
	}

	l := logging.New(callOnLogErr, callOnFail, &stdoutwriter.Logger{})
	ctx, cancel := context.WithCancel(context.Background())
	verifier := wallet.NewVerifier()
	signer, err := wallet.New()
	assert.NilError(t, err)
	_, err = NewAccountingBook(ctx, Config{}, verifier, &signer, l)
	assert.NilError(t, err)
	cancel()
	time.Sleep(time.Millisecond * 200)
}
