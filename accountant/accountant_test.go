package accountant

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
	ctx, cancel := context.WithCancel(context.Background())

	l := logging.New(callOnLogErr, callOnFail, &stdoutwriter.Logger{})
	verifier := wallet.NewVerifier()
	signer, err := wallet.New()
	assert.NilError(t, err)
	_, err = NewAccountingBook(ctx, Config{}, verifier, &signer, l)
	assert.NilError(t, err)
	time.Sleep(time.Millisecond * 200)
	cancel()
	time.Sleep(time.Millisecond * 200)
}

func TestNewVertex(t *testing.T) {
	signer, err := wallet.New()
	assert.NilError(t, err)
	trx, err := transaction.New("Vertex Test", spice.New(10, 10), []byte{}, signer.Address(), &signer)
	assert.NilError(t, err)
	_, err = NewVertex(trx, [32]byte{}, [32]byte{}, &signer)
	assert.NilError(t, err)
}

func TestCreateGensis(t *testing.T) {
	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Faield with error: %s\n", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := logging.New(callOnLogErr, callOnFail, &stdoutwriter.Logger{})
	verifier := wallet.NewVerifier()
	signer, err := wallet.New()
	assert.NilError(t, err)
	ab, err := NewAccountingBook(ctx, Config{}, verifier, &signer, l)
	assert.NilError(t, err)

	genesisSpice := spice.New(math.MaxUint64-1, 1000000000000000000)

	receiver, err := wallet.New()
	assert.NilError(t, err)
	vrx, err := ab.CreateGenesis("GENESIS", genesisSpice, []byte{}, &receiver)
	assert.NilError(t, err)
	ok := vrx.Transaction.IsSpiceTransfer()
	assert.Equal(t, ok, true)
	assert.DeepEqual(t, genesisSpice, vrx.Transaction.Spice)
}

func TestSingleIssuerSingleReceiverSpiceTransfer(t *testing.T) {
	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Faield with error: %s\n", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := logging.New(callOnLogErr, callOnFail, &stdoutwriter.Logger{})
	verifier := wallet.NewVerifier()
	signer, err := wallet.New()
	assert.NilError(t, err)
	ab, err := NewAccountingBook(ctx, Config{}, verifier, &signer, l)
	assert.NilError(t, err)

	genesisSpice := spice.New(math.MaxUint64-1, 0)

	genesisReceiver, err := wallet.New()
	assert.NilError(t, err)
	vrx, err := ab.CreateGenesis("GENESIS", genesisSpice, []byte{}, &genesisReceiver)
	assert.NilError(t, err)
	ok := vrx.Transaction.IsSpiceTransfer()
	assert.Equal(t, ok, true)
	assert.DeepEqual(t, genesisSpice, vrx.Transaction.Spice)

	receiver, err := wallet.New()
	assert.NilError(t, err)
	spiceMainTransfer := 10
	var mainSpiceReduction uint64
	for i := 0; i < 100; i++ {
		spc := spice.New(uint64(spiceMainTransfer), 0)
		trx, err := transaction.New(fmt.Sprintf("Spice supply %v", i), spc, []byte{}, receiver.Address(), &genesisReceiver)
		assert.NilError(t, err)
		_, err = ab.CreateLeaf(ctx, &trx)
		assert.NilError(t, err)
		mainSpiceReduction += uint64(spiceMainTransfer)
	}

	balance, err := ab.CalculateBalance(ctx, receiver.Address())
	assert.NilError(t, err)
	assert.Equal(t, balance.Spice.Currency, mainSpiceReduction-uint64(spiceMainTransfer))

	time.Sleep(time.Millisecond * 200)
}

func BenchmarkSingleIssuerSingleReceiverSpiceTransfer(b *testing.B) {
	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Faield with error: %s\n", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := logging.New(callOnLogErr, callOnFail, &stdoutwriter.Logger{})
	verifier := wallet.NewVerifier()
	signer, err := wallet.New()
	assert.NilError(b, err)
	ab, err := NewAccountingBook(ctx, Config{}, verifier, &signer, l)
	assert.NilError(b, err)

	genesisSpice := spice.New(math.MaxUint64-1, 0)

	genesisReceiver, err := wallet.New()
	assert.NilError(b, err)
	vrx, err := ab.CreateGenesis("GENESIS", genesisSpice, []byte{}, &genesisReceiver)
	assert.NilError(b, err)
	ok := vrx.Transaction.IsSpiceTransfer()
	assert.Equal(b, ok, true)
	assert.DeepEqual(b, genesisSpice, vrx.Transaction.Spice)

	receiver, err := wallet.New()
	assert.NilError(b, err)
	spiceMainTransfer := 10
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		spc := spice.New(uint64(spiceMainTransfer), 0)
		trx, err := transaction.New(fmt.Sprintf("Spice supply %v", n), spc, []byte{}, receiver.Address(), &genesisReceiver)
		assert.NilError(b, err)
		_, err = ab.CreateLeaf(ctx, &trx)
		assert.NilError(b, err)
	}
}
