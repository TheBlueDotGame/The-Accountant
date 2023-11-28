package cache

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/bartossh/Computantis/src/spice"
	"github.com/bartossh/Computantis/src/transaction"
	"github.com/bartossh/Computantis/src/wallet"
	"gotest.tools/v3/assert"
)

const (
	maxCacheSizeMB = 1024
	maxEntrySize   = 32 * 10_000
)

func createRandomTransaction(signer, receiver *wallet.Wallet, r *rand.Rand) (transaction.Transaction, error) {
	data := make([]byte, 32)
	r.Read(data)

	return transaction.New("Test cache", spice.Melange{}, data, receiver.Address(), signer)
}

func TestSaveToCacheSuccess(t *testing.T) {
	w, err := wallet.New()
	assert.NilError(t, err)
	wr, err := wallet.New()
	assert.NilError(t, err)

	r := rand.New(rand.NewSource(time.Now().Unix()))

	hippo, err := New(maxEntrySize, maxCacheSizeMB)
	assert.NilError(t, err)

	trxNum := 1000
	for i := 0; i < trxNum; i++ {
		trx, err := createRandomTransaction(&w, &wr, r)
		assert.NilError(t, err)

		err = hippo.SaveAwaitedTransaction(&trx)
		assert.NilError(t, err)
	}
}

func TestReadFromCacheSuccess(t *testing.T) {
	w, err := wallet.New()
	assert.NilError(t, err)
	wr, err := wallet.New()
	assert.NilError(t, err)

	r := rand.New(rand.NewSource(time.Now().Unix()))

	hippo, err := New(maxEntrySize, maxCacheSizeMB)
	assert.NilError(t, err)

	trxNum := 100
	hashes := make(map[[32]byte]struct{}, trxNum)
	for i := 0; i < trxNum; i++ {
		trx, err := createRandomTransaction(&w, &wr, r)
		assert.NilError(t, err)
		hashes[trx.Hash] = struct{}{}

		err = hippo.SaveAwaitedTransaction(&trx)
		assert.NilError(t, err)
	}

	trxs, err := hippo.ReadTransactions(w.Address())
	assert.NilError(t, err)
	assert.Equal(t, len(trxs), trxNum)
	for _, trx := range trxs {
		_, ok := hashes[trx.Hash]
		assert.Equal(t, ok, true)
	}
}

func TestReadFromCacheSucessEdgeCaseIssuerAndReceiverTheSame(t *testing.T) {
	w, err := wallet.New()
	assert.NilError(t, err)

	r := rand.New(rand.NewSource(time.Now().Unix()))

	hippo, err := New(maxEntrySize, maxCacheSizeMB)
	assert.NilError(t, err)

	trxNum := 100
	hashes := make(map[[32]byte]struct{}, trxNum)
	for i := 0; i < trxNum; i++ {
		trx, err := createRandomTransaction(&w, &w, r)
		assert.NilError(t, err)
		hashes[trx.Hash] = struct{}{}

		err = hippo.SaveAwaitedTransaction(&trx)
		assert.NilError(t, err)
	}
	trxs, err := hippo.ReadTransactions(w.Address())
	assert.NilError(t, err)
	assert.Equal(t, len(trxs), trxNum)
	for _, trx := range trxs {
		_, ok := hashes[trx.Hash]
		assert.Equal(t, ok, true)
	}
}

func BenchmarkReadCacheSuccess(b *testing.B) {
	trxToSave := []int{1, 10, 100, 1000}
	for _, trxNum := range trxToSave {
		b.Run(fmt.Sprintf("test case %v", trxNum), func(b *testing.B) {
			w, err := wallet.New()
			assert.NilError(b, err)
			wr, err := wallet.New()
			assert.NilError(b, err)

			r := rand.New(rand.NewSource(time.Now().Unix()))

			hippo, err := New(maxEntrySize, maxCacheSizeMB)
			assert.NilError(b, err)
			for i := 0; i < trxNum; i++ {
				b.StopTimer()
				trx, err := createRandomTransaction(&w, &wr, r)
				assert.NilError(b, err)
				b.StartTimer()

				err = hippo.SaveAwaitedTransaction(&trx)
				assert.NilError(b, err)
			}
			b.ResetTimer()

			for n := 0; n < b.N; n++ {
				_, err := hippo.ReadTransactions(w.Address())
				assert.NilError(b, err)
			}
		})
	}
}

func TestRemoveFromCacheSucessEdgeCaseIssuerAndReceiverTheSame(t *testing.T) {
	w, err := wallet.New()
	assert.NilError(t, err)

	r := rand.New(rand.NewSource(time.Now().Unix()))

	hippo, err := New(maxEntrySize, maxCacheSizeMB)
	assert.NilError(t, err)

	trxNum := 100
	trxHases := make([][32]byte, 0, trxNum)
	for i := 0; i < trxNum; i++ {
		trx, err := createRandomTransaction(&w, &w, r)
		assert.NilError(t, err)

		trxHases = append(trxHases, trx.Hash)

		err = hippo.SaveAwaitedTransaction(&trx)
		assert.NilError(t, err)
	}
	trxs, err := hippo.ReadTransactions(w.Address())
	assert.NilError(t, err)
	assert.Equal(t, len(trxs), trxNum)

	for _, hash := range trxHases {
		_, err := hippo.RemoveAwaitedTransaction(hash, w.Address())
		assert.NilError(t, err)
	}

	trxs, err = hippo.ReadTransactions(w.Address())
	assert.NilError(t, err)
	assert.Equal(t, len(trxs), 0)
}

func TestRemoveFromCacheFailureWrongAddress(t *testing.T) {
	w, err := wallet.New()
	assert.NilError(t, err)
	wr, err := wallet.New()
	assert.NilError(t, err)

	r := rand.New(rand.NewSource(time.Now().Unix()))

	hippo, err := New(maxEntrySize, maxCacheSizeMB)
	assert.NilError(t, err)

	trxNum := 100
	trxHases := make([][32]byte, 0, trxNum)
	for i := 0; i < trxNum; i++ {
		trx, err := createRandomTransaction(&w, &wr, r) // wr is a receiver
		assert.NilError(t, err)

		trxHases = append(trxHases, trx.Hash)

		err = hippo.SaveAwaitedTransaction(&trx)
		assert.NilError(t, err)
	}
	trxs, err := hippo.ReadTransactions(w.Address())
	assert.NilError(t, err)
	assert.Equal(t, len(trxs), trxNum)

	for _, hash := range trxHases {
		_, err := hippo.RemoveAwaitedTransaction(hash, w.Address()) // try to remove by issuer impossible
		assert.ErrorContains(t, err, ErrUnauthorized.Error())
	}

	trxs, err = hippo.ReadTransactions(w.Address())
	assert.NilError(t, err)
	assert.Equal(t, len(trxs), trxNum)
}
