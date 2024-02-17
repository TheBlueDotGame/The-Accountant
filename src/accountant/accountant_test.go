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

	"github.com/bartossh/Computantis/src/logging"
	"github.com/bartossh/Computantis/src/spice"
	"github.com/bartossh/Computantis/src/stdoutwriter"
	"github.com/bartossh/Computantis/src/transaction"
	"github.com/bartossh/Computantis/src/wallet"
	msgpackv2 "github.com/shamaton/msgpack/v2"
	"github.com/vmihailenco/msgpack"
	"gotest.tools/v3/assert"
)

type EmptyLogger struct{}

func (l EmptyLogger) Write(p []byte) (n int, err error) {
	return len(p), nil
}

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
		Weight:          1,
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
		Weight:          1,
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
	_, err = NewVertex(trx, [32]byte{}, [32]byte{}, 0, &signer)
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
	vrx, err := ab.CreateGenesis("GENESIS", genesisSpice, []byte{}, receiver.Address())
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
	vrx, err := ab.CreateGenesis("GENESIS", genesisSpice, []byte{}, genesisReceiver.Address())
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
	assert.Equal(t, balance.Spice.Currency, mainSpiceReduction)

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
	vrx, err := ab.CreateGenesis("GENESIS", genesisSpice, []byte{}, genesisReceiver.Address())
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
	assert.Equal(t, int64(balance.Spice.Currency), mainSpiceReduction.Load())

	time.Sleep(time.Millisecond * 200)
}

func TestMultipleIssuerMultipleReceiversSpiceTransferConcurentLegitimate(t *testing.T) {
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
	vrx, err := ab.CreateGenesis("GENESIS", genesisSpice, []byte{}, genesisReceiver.Address())
	assert.NilError(t, err)
	ok := vrx.Transaction.IsSpiceTransfer()
	assert.Equal(t, ok, true)
	assert.DeepEqual(t, genesisSpice, vrx.Transaction.Spice)

	issuer := genesisReceiver
	numberOfParticipants := 200
	numberOfRounds := 10

	for rec := 0; rec < numberOfParticipants; rec++ {
		receiver, err := wallet.New()
		assert.NilError(t, err)
		spiceMainTransfer := 10
		var mainSpiceReduction atomic.Int64
		var wg sync.WaitGroup
		for i := 0; i < numberOfRounds; i++ {
			wg.Add(1)
			go func(next int, wg *sync.WaitGroup) {
				spc := spice.New(uint64(spiceMainTransfer), 0)
				trx, err := transaction.New(
					fmt.Sprintf("Spice supply from: %v, trx number: %v", rec, next),
					spc, []byte{}, receiver.Address(), &issuer,
				)
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
		assert.Equal(t, int64(balance.Spice.Currency), mainSpiceReduction.Load())
		issuer = receiver
	}

	time.Sleep(time.Millisecond * 200)
}

func TestMultipleIssuerMultipleReceiversSpiceTransferConcurentDoubleSpending(t *testing.T) {
	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Failed with error: %s\n", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := logging.New(callOnLogErr, callOnFail, &EmptyLogger{})
	verifier := wallet.NewVerifier()
	signer, err := wallet.New()
	assert.NilError(t, err)
	ab, err := NewAccountingBook(ctx, Config{}, verifier, &signer, l)
	assert.NilError(t, err)

	genesisSpice := spice.New(math.MaxUint64-1, 0)

	genesisReceiver, err := wallet.New()
	assert.NilError(t, err)
	vrx, err := ab.CreateGenesis("GENESIS", genesisSpice, []byte{}, genesisReceiver.Address())
	assert.NilError(t, err)
	ok := vrx.Transaction.IsSpiceTransfer()
	assert.Equal(t, ok, true)
	assert.DeepEqual(t, genesisSpice, vrx.Transaction.Spice)

	numberOfParticipants := 100
	roundsPerPArticipant := 10
	spiceMainTransfer := 10
	issuer := genesisReceiver
	walletsToCheck := make([]transaction.Signer, 0, numberOfParticipants)

	for rec := 0; rec < numberOfParticipants; rec++ {
		receiver, err := wallet.New()
		assert.NilError(t, err)
		spiceMainTransfer := 10
		var mainSpiceReduction atomic.Int64
		var wg sync.WaitGroup
		for i := 0; i < roundsPerPArticipant; i++ {
			wg.Add(1)
			go func(next int, wg *sync.WaitGroup) {
				spc := spice.New(uint64(spiceMainTransfer), 0)
				trx, err := transaction.New(fmt.Sprintf("Spice supply from: %v, trx number: %v", rec, next), spc, []byte{}, receiver.Address(), &issuer)
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
		issuer = receiver
		walletsToCheck = append(walletsToCheck, &receiver)
	}
	for idx, issuer := range walletsToCheck {
		receiver, err := wallet.New()
		assert.NilError(t, err)
		var mainSpiceReduction atomic.Int64
		var wg sync.WaitGroup
		for i := 0; i < roundsPerPArticipant; i++ {
			wg.Add(1)
			go func(next int, wg *sync.WaitGroup) {
				spc := spice.New(uint64(spiceMainTransfer), 0)
				trx, err := transaction.New(fmt.Sprintf(
					"Spice supply from: %v, to %v, next %v\n",
					issuer.Address(), receiver.Address(), next),
					spc, []byte{}, receiver.Address(), issuer,
				)
				assert.NilError(t, err)
				_, err = ab.CreateLeaf(ctx, &trx)
				assert.NilError(t, err)
				mainSpiceReduction.Add(int64(spiceMainTransfer))
				wg.Done()
			}(i, &wg)
		}

		wg.Wait()
		balance, err := ab.CalculateBalance(ctx, issuer.Address())
		assert.NilError(t, err)
		switch idx {
		case numberOfParticipants - 1:
			assert.Equal(t, balance.Spice.Currency, uint64(spiceMainTransfer))
		default:
			assert.Equal(t, balance.Spice.Currency, uint64(0))
		}
	}

	time.Sleep(time.Millisecond * 200)
}

func TestMultipleIssuerMultipleReceiversMultipleAccountantSpiceTransferLegitimate(t *testing.T) {
	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Failed with error: %s\n", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := logging.New(callOnLogErr, callOnFail, &stdoutwriter.Logger{})

	numberOfNodes := 10
	nodes := make([]*AccountingBook, 0, numberOfNodes)
	var vrx Vertex
	var issuer wallet.Wallet
	for i := 0; i < numberOfNodes; i++ {
		verifier := wallet.NewVerifier()
		signer, err := wallet.New()
		assert.NilError(t, err)
		ab, err := NewAccountingBook(ctx, Config{}, verifier, &signer, l)
		assert.NilError(t, err)

		switch i {
		case 0:
			genesisSpice := spice.New(math.MaxUint64-1, 0)
			genesisReceiver, err := wallet.New()
			assert.NilError(t, err)
			vrx, err = ab.CreateGenesis("GENESIS", genesisSpice, []byte{}, genesisReceiver.Address())
			assert.NilError(t, err)
			ok := vrx.Transaction.IsSpiceTransfer()
			assert.Equal(t, ok, true)
			assert.DeepEqual(t, genesisSpice, vrx.Transaction.Spice)
			issuer = genesisReceiver
		default:
			ctxx, cancelF := context.WithCancelCause(ctx)
			cVrx := make(chan *Vertex, 2)
			go ab.LoadDag(ctxx, cancelF, cVrx) // NOTE: crucial for tests, on all node genesis nodes dag shall be loaded from genessis
			cVrx <- &vrx
			time.Sleep(time.Millisecond * 100)
			cancelF(nil)
		}

		nodes = append(nodes, ab)
	}

	numberOfParticipants := 2
	numberOfRounds := 4

	for rec := 0; rec < numberOfParticipants; rec++ {
		receiver, err := wallet.New()
		assert.NilError(t, err)
		spiceMainTransfer := 10
		for i := 0; i < numberOfRounds; i++ {
			spc := spice.New(uint64(spiceMainTransfer), 0)
			trx, err := transaction.New(fmt.Sprintf("Spice supply from: %v to %v, trx number: %v", issuer.Address(), receiver.Address(), i), spc, []byte{}, receiver.Address(), &issuer)
			assert.NilError(t, err)
			nodeNum := rand.Intn(len(nodes))
			ab := nodes[nodeNum]
			leaf, err := ab.CreateLeaf(ctx, &trx)
			assert.NilError(t, err)

			for idx := range nodes {
				if idx == nodeNum {
					continue
				}
				err := nodes[idx].AddLeaf(ctx, &leaf)
				assert.NilError(t, err)
			}
			spiceIssuerStart := spice.New(math.MaxUint64-1, 0)
			spiceReceiverStart := spice.New(0, 0)
			for _, ab := range nodes {
				balanceIss, err := ab.CalculateBalance(ctx, issuer.Address())
				assert.NilError(t, err)
				assert.Equal(t, balanceIss.Spice.Currency < spiceIssuerStart.Currency, true)
				balanceRec, err := ab.CalculateBalance(ctx, receiver.Address())
				assert.NilError(t, err)
				assert.Equal(t, balanceRec.Spice.Currency > spiceReceiverStart.Currency, true)
			}
		}

		issuer = receiver // exchange to pour founds over
	}

	time.Sleep(time.Millisecond * 200)
}

func TestMultipleIssuerMultipleReceiversMultipleAccountantSpiceLoadDAG(t *testing.T) {
	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Failed with error: %s\n", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := logging.New(callOnLogErr, callOnFail, &stdoutwriter.Logger{})

	var vrx Vertex
	var issuer wallet.Wallet
	verifier := wallet.NewVerifier()
	signer, err := wallet.New()
	assert.NilError(t, err)
	ab, err := NewAccountingBook(ctx, Config{}, verifier, &signer, l)
	assert.NilError(t, err)

	genesisSpice := spice.New(math.MaxUint64-1, 0)
	genesisReceiver, err := wallet.New()
	assert.NilError(t, err)
	vrx, err = ab.CreateGenesis("GENESIS", genesisSpice, []byte{}, genesisReceiver.Address())
	assert.NilError(t, err)
	ok := vrx.Transaction.IsSpiceTransfer()
	assert.Equal(t, ok, true)
	assert.DeepEqual(t, genesisSpice, vrx.Transaction.Spice)
	issuer = genesisReceiver

	numberOfParticipants := 20
	numberOfRounds := 10

	var receiver wallet.Wallet

	for rec := 0; rec < numberOfParticipants; rec++ {
		receiver, err := wallet.New()
		assert.NilError(t, err)
		spiceMainTransfer := 10
		var mainSpiceReduction atomic.Int64
		for i := 0; i < numberOfRounds; i++ {
			spc := spice.New(uint64(spiceMainTransfer), 0)
			trx, err := transaction.New(fmt.Sprintf("Spice supply from: %v to %v, trx number: %v", issuer.Address(), receiver.Address(), i), spc, []byte{}, receiver.Address(), &issuer)
			assert.NilError(t, err)
			_, err = ab.CreateLeaf(ctx, &trx)
			assert.NilError(t, err)
			mainSpiceReduction.Add(int64(spiceMainTransfer))
		}
		issuer = receiver
	}

	balanceGenessis, err := ab.CalculateBalance(ctx, receiver.Address())
	assert.NilError(t, err)
	// Load Dag test.
	abLoad, err := NewAccountingBook(ctx, Config{}, verifier, &signer, l)
	assert.NilError(t, err)

	ctxx, cancelF := context.WithCancelCause(ctx)
	cVrx, cErr := ab.StreamDAG(ctxx)
	go abLoad.LoadDag(ctxx, cancelF, cVrx)

	err = <-cErr
	assert.NilError(t, err)
	cancelF(nil)

	balanceLoadedDag, err := ab.CalculateBalance(ctx, receiver.Address())
	assert.NilError(t, err)

	time.Sleep(time.Millisecond * 200)
	assert.Equal(t, int64(balanceGenessis.Spice.Currency), int64(balanceLoadedDag.Spice.Currency))
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
	vrx, err := ab.CreateGenesis("GENESIS", genesisSpice, []byte{}, genesisReceiver.Address())
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
		trx, err := transaction.New(fmt.Sprintf("First spice supply %v", i), spc, []byte{}, receiver.Address(), &genesisReceiver)
		assert.NilError(b, err)
		_, err = ab.CreateLeaf(ctx, &trx)
		assert.NilError(b, err)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		spc := spice.New(uint64(spiceMainTransfer), 0)
		trx, err := transaction.New(fmt.Sprintf("Second spice supply %v", n), spc, []byte{}, receiver.Address(), &genesisReceiver)
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
	vrx, err := ab.CreateGenesis("GENESIS", genesisSpice, []byte{}, genesisReceiver.Address())
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
		trx, err := transaction.New(fmt.Sprintf("First spice supply %v", i), spc, []byte{}, receiver.Address(), &genesisReceiver)
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
			trx, err := transaction.New(fmt.Sprintf("Second spice supply %v", next), spc, []byte{}, receiver.Address(), &genesisReceiver)
			assert.NilError(b, err)
			_, err = ab.CreateLeaf(ctx, &trx)
			assert.NilError(b, err)
			wg.Done()
		}(n, &wg)
	}
	wg.Wait()
}

func BenchmarkMultipleIssuerMultipleReceiversSpiceTransferConcurentLegitimate(b *testing.B) {
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
	vrx, err := ab.CreateGenesis("GENESIS", genesisSpice, []byte{}, genesisReceiver.Address())
	assert.NilError(b, err)
	ok := vrx.Transaction.IsSpiceTransfer()
	assert.Equal(b, ok, true)
	assert.DeepEqual(b, genesisSpice, vrx.Transaction.Spice)

	issuer := genesisReceiver
	// Pre-populate
	spiceMainTransfer := 10
	for i := 0; i < 1000; i++ {
		receiver, err := wallet.New()
		assert.NilError(b, err)
		spc := spice.New(uint64(spiceMainTransfer), 0)
		trx, err := transaction.New(fmt.Sprintf("First spice supply %v", i), spc, []byte{}, receiver.Address(), &genesisReceiver)
		assert.NilError(b, err)
		_, err = ab.CreateLeaf(ctx, &trx)
		assert.NilError(b, err)
	}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		b.StopTimer()
		receiver, err := wallet.New()
		assert.NilError(b, err)
		var mainSpiceReduction atomic.Int64
		b.StartTimer()
		spc := spice.New(uint64(spiceMainTransfer), 0)
		trx, err := transaction.New(fmt.Sprintf("Spice supply from rec: %s to issuer %s", issuer.Address(), receiver.Address()), spc, []byte{}, receiver.Address(), &issuer)
		assert.NilError(b, err)
		_, err = ab.CreateLeaf(ctx, &trx)
		assert.NilError(b, err)
		mainSpiceReduction.Add(int64(spiceMainTransfer))
		issuer = receiver
	}
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

func TestVertexStorageAddRepeted(t *testing.T) {
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

func TestTrxToVrxStorageSaveReadRemove(t *testing.T) {
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

	err = ab.removeTrxInVertex(trxHash)
	assert.NilError(t, err)

	ok, err = ab.checkTrxInVertexExists(trxHash)
	assert.NilError(t, err)
	assert.Equal(t, ok, false)
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
