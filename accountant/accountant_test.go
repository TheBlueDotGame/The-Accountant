package accountant

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
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

func TestSingleIssuerSingleReceiverSpiceTransferConsecutive(t *testing.T) {
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

func TestSingleIssuerSingleReceiverSpiceTransferConcurent(t *testing.T) {
	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Failed with error: %s\n", err)
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
	var mainSpiceReduction atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(next int, wg *sync.WaitGroup) {
			spc := spice.New(uint64(spiceMainTransfer), 0)
			trx, err := transaction.New(fmt.Sprintf("Spice supply %v", next), spc, []byte{}, receiver.Address(), &genesisReceiver)
			assert.NilError(t, err)
			_, err = ab.CreateLeaf(ctx, &trx)
			assert.NilError(t, err)
			mainSpiceReduction.Add(int64(spiceMainTransfer))
			wg.Done()
		}(i, &wg)
	}

	wg.Wait()

	balance, err := ab.CalculateBalance(ctx, receiver.Address())
	assert.NilError(t, err)
	assert.Equal(t, int64(balance.Spice.Currency), mainSpiceReduction.Add(int64(-spiceMainTransfer)))

	time.Sleep(time.Millisecond * 200)
}

func BenchmarkSingleIssuerSingleReceiverSpiceTransferConsecutive(b *testing.B) {
	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Failed with error: %s\n", err)
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
	// Pre-populate
	for i := 0; i < 1000; i++ {
		spc := spice.New(uint64(spiceMainTransfer), 0)
		trx, err := transaction.New(fmt.Sprintf("Spice supply %v", i), spc, []byte{}, receiver.Address(), &genesisReceiver)
		assert.NilError(b, err)
		_, err = ab.CreateLeaf(ctx, &trx)
		assert.NilError(b, err)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		spc := spice.New(uint64(spiceMainTransfer), 0)
		trx, err := transaction.New(fmt.Sprintf("Spice supply %v", n), spc, []byte{}, receiver.Address(), &genesisReceiver)
		assert.NilError(b, err)
		_, err = ab.CreateLeaf(ctx, &trx)
		assert.NilError(b, err)
	}
}

func BenchmarkSingleIssuerSingleReceiverSpiceTransferConcurrent(b *testing.B) {
	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Failed with error: %s\n", err)
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
	// Pre-populate
	for i := 0; i < 1000; i++ {
		spc := spice.New(uint64(spiceMainTransfer), 0)
		trx, err := transaction.New(fmt.Sprintf("Spice supply %v", i), spc, []byte{}, receiver.Address(), &genesisReceiver)
		assert.NilError(b, err)
		_, err = ab.CreateLeaf(ctx, &trx)
		assert.NilError(b, err)
	}
	b.ResetTimer()
	var wg sync.WaitGroup
	for n := 0; n < b.N; n++ {
		wg.Add(1)
		go func(next int, wg *sync.WaitGroup) {
			spc := spice.New(uint64(spiceMainTransfer), 0)
			trx, err := transaction.New(fmt.Sprintf("Spice supply %v", next), spc, []byte{}, receiver.Address(), &genesisReceiver)
			assert.NilError(b, err)
			_, err = ab.CreateLeaf(ctx, &trx)
			assert.NilError(b, err)
			wg.Done()
		}(n, &wg)
	}
	wg.Wait()
}

func TestVertexStorageAdd(t *testing.T) {
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

	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Failed with error: %s\n", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := logging.New(callOnLogErr, callOnFail, &stdoutwriter.Logger{})
	verifier := wallet.NewVerifier()
	signer, err := wallet.New()
	assert.NilError(t, err)
	ab, err := NewAccountingBook(ctx, Config{}, verifier, &signer, l)
	assert.NilError(t, err)

	err = ab.saveVertexToStorage(&v)
	assert.NilError(t, err)
}

func TestVertexStorageAddRepret(t *testing.T) {
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

	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Failed with error: %s\n", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := logging.New(callOnLogErr, callOnFail, &stdoutwriter.Logger{})
	verifier := wallet.NewVerifier()
	signer, err := wallet.New()
	assert.NilError(t, err)
	ab, err := NewAccountingBook(ctx, Config{}, verifier, &signer, l)
	assert.NilError(t, err)

	err = ab.saveVertexToStorage(&v)
	assert.NilError(t, err)
	err = ab.saveVertexToStorage(&v)
	assert.ErrorIs(t, err, ErrVertexAlreadyExists)
}

func TestVertexStorageAddRead(t *testing.T) {
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

	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Failed with error: %s\n", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := logging.New(callOnLogErr, callOnFail, &stdoutwriter.Logger{})
	verifier := wallet.NewVerifier()
	signer, err := wallet.New()
	assert.NilError(t, err)
	ab, err := NewAccountingBook(ctx, Config{}, verifier, &signer, l)
	assert.NilError(t, err)

	err = ab.saveVertexToStorage(&v)
	assert.NilError(t, err)

	newV, err := ab.readVertexFromStorage(v.Hash[:])
	assert.NilError(t, err)
	assert.DeepEqual(t, newV, v)
}

func BenchmarkVertexStorageAdd(b *testing.B) {
	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Failed with error: %s\n", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := logging.New(callOnLogErr, callOnFail, &stdoutwriter.Logger{})
	verifier := wallet.NewVerifier()
	signer, err := wallet.New()
	assert.NilError(b, err)
	ab, err := NewAccountingBook(ctx, Config{}, verifier, &signer, l)
	assert.NilError(b, err)

	for n := 0; n < b.N; n++ {
		b.StopTimer()
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
		b.StartTimer()
		ab.saveVertexToStorage(&v)
	}
}

func BenchmarkVertexStorageSaveRead(b *testing.B) {
	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Failed with error: %s\n", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := logging.New(callOnLogErr, callOnFail, &stdoutwriter.Logger{})
	verifier := wallet.NewVerifier()
	signer, err := wallet.New()
	assert.NilError(b, err)
	ab, err := NewAccountingBook(ctx, Config{}, verifier, &signer, l)
	assert.NilError(b, err)

	for n := 0; n < b.N; n++ {
		b.StopTimer()
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
		b.StartTimer()
		ab.saveVertexToStorage(&v)
		ab.readVertexFromStorage(v.Hash[:])
	}
}

func TestTrxToVrxStorageSaveRead(t *testing.T) {
	trxHash := generateData(32)
	vrxHash := generateData(32)

	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Failed with error: %s\n", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := logging.New(callOnLogErr, callOnFail, &stdoutwriter.Logger{})
	verifier := wallet.NewVerifier()
	signer, err := wallet.New()
	assert.NilError(t, err)
	ab, err := NewAccountingBook(ctx, Config{}, verifier, &signer, l)
	assert.NilError(t, err)

	err = ab.saveTrxInVertex(trxHash, vrxHash)
	assert.NilError(t, err)

	ok, err := ab.checkTrxInVertexExists(trxHash)
	assert.NilError(t, err)
	assert.Equal(t, ok, true)
}

func TestTrxToVrxStorageSaveExisting(t *testing.T) {
	trxHash := generateData(32)
	vrxHash := generateData(32)

	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Failed with error: %s\n", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := logging.New(callOnLogErr, callOnFail, &stdoutwriter.Logger{})
	verifier := wallet.NewVerifier()
	signer, err := wallet.New()
	assert.NilError(t, err)
	ab, err := NewAccountingBook(ctx, Config{}, verifier, &signer, l)
	assert.NilError(t, err)

	err = ab.saveTrxInVertex(trxHash, vrxHash)
	assert.NilError(t, err)

	err = ab.saveTrxInVertex(trxHash, vrxHash)
	assert.ErrorIs(t, err, ErrTrxInVertexAlreadyExists)
}

func BenchmarkTrxToVertexStorageSave(b *testing.B) {
	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Failed with error: %s\n", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := logging.New(callOnLogErr, callOnFail, &stdoutwriter.Logger{})
	verifier := wallet.NewVerifier()
	signer, err := wallet.New()
	assert.NilError(b, err)
	ab, err := NewAccountingBook(ctx, Config{}, verifier, &signer, l)
	assert.NilError(b, err)

	for n := 0; n < b.N; n++ {
		b.StopTimer()
		trxHash := generateData(32)
		vrxHash := generateData(32)
		b.StartTimer()
		ab.saveTrxInVertex(trxHash, vrxHash)
	}
}

func BenchmarkTrxToVertexStorageSaveRead(b *testing.B) {
	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Failed with error: %s\n", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := logging.New(callOnLogErr, callOnFail, &stdoutwriter.Logger{})
	verifier := wallet.NewVerifier()
	signer, err := wallet.New()
	assert.NilError(b, err)
	ab, err := NewAccountingBook(ctx, Config{}, verifier, &signer, l)
	assert.NilError(b, err)

	for n := 0; n < b.N; n++ {
		b.StopTimer()
		trxHash := generateData(32)
		vrxHash := generateData(32)
		b.StartTimer()
		ab.saveTrxInVertex(trxHash, vrxHash)
		ab.checkTrxInVertexExists(trxHash)
	}
}
