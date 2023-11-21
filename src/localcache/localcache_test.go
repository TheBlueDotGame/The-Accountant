package localcache

import (
	crand "crypto/rand"
	"fmt"
	"math/rand"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/bartossh/Computantis/src/transaction"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func generateHash() [32]byte {
	var buf [32]byte
	crand.Read(buf[:])
	return buf
}

type cacheTsxTestCase struct {
	issuer       string
	receiver     string
	transactions []transaction.Transaction
}

func prepareTestCases(numCases, numTransactions int) []cacheTsxTestCase {
	cases := make([]cacheTsxTestCase, 0, numCases)
	for i := 0; i < numCases; i++ {
		c := cacheTsxTestCase{
			issuer:       randStringRunes(32),
			receiver:     randStringRunes(32),
			transactions: make([]transaction.Transaction, 0, numTransactions),
		}
		for j := 0; j < numTransactions; j++ {
			c.transactions = append(c.transactions, transaction.Transaction{
				Hash:            generateHash(),
				IssuerAddress:   c.issuer,
				ReceiverAddress: c.receiver,
			})
		}
		cases = append(cases, c)

	}

	return cases
}

func TestCorrectTransactionInsert(t *testing.T) {
	cache := NewTransactionCache(Config{
		MaxLen: 10000,
	})
	cases := prepareTestCases(100, 100)
	for _, c := range cases {
		for _, trx := range c.transactions {
			err := cache.WriteIssuerSignedTransactionForReceiver(&trx)
			assert.NilError(t, err)
		}
	}
}

func TestNotEnoughSpaceTransactionInsert(t *testing.T) {
	cache := NewTransactionCache(Config{
		MaxLen: 9999,
	})
	cases := prepareTestCases(100, 100)
	counter := 9999
	for _, c := range cases {
		for _, trx := range c.transactions {
			err := cache.WriteIssuerSignedTransactionForReceiver(&trx)
			counter--
			if counter >= 0 {
				assert.NilError(t, err)
				continue
			}
			assert.ErrorContains(t, err, "cannot add to cache")
		}
	}
}

func TestCorrectTransactionReadIssuer(t *testing.T) {
	cache := NewTransactionCache(Config{
		MaxLen: 10000,
	})
	cases := prepareTestCases(100, 100)
	for _, c := range cases {
		for _, trx := range c.transactions {
			err := cache.WriteIssuerSignedTransactionForReceiver(&trx)
			assert.NilError(t, err)
		}
	}

	for i, c := range cases {
		trxs, err := cache.ReadAwaitingTransactionsByIssuer(c.issuer)
		assert.NilError(t, err, fmt.Sprintf("case index: %v", i))
		assert.Equal(t, len(trxs), 100)
	}
}

func TestCorrectTransactionReadReceiver(t *testing.T) {
	cache := NewTransactionCache(Config{
		MaxLen: 10000,
	})
	cases := prepareTestCases(100, 100)
	for _, c := range cases {
		for _, trx := range c.transactions {
			err := cache.WriteIssuerSignedTransactionForReceiver(&trx)
			assert.NilError(t, err)
		}
	}

	for i, c := range cases {
		trxs, err := cache.ReadAwaitingTransactionsByReceiver(c.receiver)
		assert.NilError(t, err, fmt.Sprintf("case index: %v", i))
		assert.Equal(t, len(trxs), 100)
	}
}

func TestFailedToTransactionReadReceiverWhereIssuer(t *testing.T) {
	cache := NewTransactionCache(Config{
		MaxLen: 10000,
	})
	cases := prepareTestCases(100, 100)
	for _, c := range cases {
		for _, trx := range c.transactions {
			err := cache.WriteIssuerSignedTransactionForReceiver(&trx)
			assert.NilError(t, err)
		}
	}

	for i, c := range cases {
		trxs, err := cache.ReadAwaitingTransactionsByIssuer(c.receiver)
		assert.ErrorContains(t, err, "has no matching transaction", fmt.Sprintf("case index: %v", i))
		assert.Equal(t, len(trxs), 0)
	}
}

func TestFailedToTransactionReadIssuerWhereReceiver(t *testing.T) {
	cache := NewTransactionCache(Config{
		MaxLen: 10000,
	})
	cases := prepareTestCases(100, 100)
	for _, c := range cases {
		for _, trx := range c.transactions {
			err := cache.WriteIssuerSignedTransactionForReceiver(&trx)
			assert.NilError(t, err)
		}
	}

	for i, c := range cases {
		trxs, err := cache.ReadAwaitingTransactionsByReceiver(c.issuer)
		assert.ErrorContains(t, err, "has no matching transaction", fmt.Sprintf("case index: %v", i))
		assert.Equal(t, len(trxs), 0)
	}
}

func TestFailTransactionAddSameHash(t *testing.T) {
	cache := NewTransactionCache(Config{
		MaxLen: 20000,
	})
	cases := prepareTestCases(100, 100)
	for _, c := range cases {
		for _, trx := range c.transactions {
			err := cache.WriteIssuerSignedTransactionForReceiver(&trx)
			assert.NilError(t, err)
		}
	}

	for _, c := range cases {
		for i, trx := range c.transactions {
			err := cache.WriteIssuerSignedTransactionForReceiver(&trx)
			assert.ErrorIs(t, err, ErrNotAlloweReoccurringHash, fmt.Sprintf("case index: %v", i))
		}
	}
}

func TestFailTransactionNotExist(t *testing.T) {
	cache := NewTransactionCache(Config{
		MaxLen: 20000,
	})
	cases := prepareTestCases(100, 100)
	for _, c := range cases {
		for _, trx := range c.transactions {
			err := cache.WriteIssuerSignedTransactionForReceiver(&trx)
			assert.NilError(t, err)
		}
	}

	for _, c := range cases {
		cache.CleanSignedTransactions(c.transactions)
	}

	for i, c := range cases {
		trxs, err := cache.ReadAwaitingTransactionsByIssuer(c.issuer)
		assert.ErrorContains(t, err, "has no matching transaction", fmt.Sprintf("case index: %v", i))
		assert.Equal(t, len(trxs), 0)
	}

	for i, c := range cases {
		trxs, err := cache.ReadAwaitingTransactionsByReceiver(c.receiver)
		assert.ErrorContains(t, err, "has no matching transaction", fmt.Sprintf("case index: %v", i))
		assert.Equal(t, len(trxs), 0)
	}
}

func TestCorrectTransactionReadReceiverTwoTrxsListsOneAdded(t *testing.T) {
	cache := NewTransactionCache(Config{
		MaxLen: 10000,
	})
	cases0 := prepareTestCases(10, 10)
	for _, c := range cases0 {
		for _, trx := range c.transactions {
			err := cache.WriteIssuerSignedTransactionForReceiver(&trx)
			assert.NilError(t, err)
		}
	}

	cases1 := prepareTestCases(10, 10)

	for i, c := range cases1 {
		trxs, err := cache.ReadAwaitingTransactionsByReceiver(c.receiver)
		assert.ErrorContains(t, err, "has no matching transaction", fmt.Sprintf("case index: %v", i))
		assert.Equal(t, len(trxs), 0)
	}
}

func TestCorrectTransactionReadIssuerTwoTrxsListsOneAdded(t *testing.T) {
	cache := NewTransactionCache(Config{
		MaxLen: 100000,
	})
	cases0 := prepareTestCases(10, 10)
	for _, c := range cases0 {
		for _, trx := range c.transactions {
			err := cache.WriteIssuerSignedTransactionForReceiver(&trx)
			assert.NilError(t, err)
		}
	}

	cases1 := prepareTestCases(10, 10)

	for i, c := range cases1 {
		trxs, err := cache.ReadAwaitingTransactionsByReceiver(c.issuer)
		assert.ErrorContains(t, err, "has no matching transaction", fmt.Sprintf("case index: %v", i))
		assert.Equal(t, len(trxs), 0)
	}
}

func BenchmarkInsertIncreasingWallets100TrxEach(b *testing.B) {
	walletsCount := []int{100, 1000}
	cache := NewTransactionCache(Config{
		MaxLen: 1100000,
	})
	for _, numWallets := range walletsCount {
		b.Run(fmt.Sprintf("wallet count %v", numWallets), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				b.StopTimer()
				cases := prepareTestCases(numWallets, 100)
				b.StartTimer()
				for _, c := range cases {
					for _, trx := range c.transactions {
						cache.WriteIssuerSignedTransactionForReceiver(&trx)
					}
				}
			}
		})
	}
}

func BenchmarkInsertIncreasingTrxs100WalletsEach(b *testing.B) {
	trxsCount := []int{100, 1000}
	cache := NewTransactionCache(Config{
		MaxLen: 1100000,
	})
	for _, numTrxs := range trxsCount {
		b.Run(fmt.Sprintf("trxs count %v", numTrxs), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				b.StopTimer()
				cases := prepareTestCases(100, numTrxs)
				b.StartTimer()
				for _, c := range cases {
					for _, trx := range c.transactions {
						cache.WriteIssuerSignedTransactionForReceiver(&trx)
					}
				}
			}
		})
	}
}

func BenchmarkReadIssure(b *testing.B) {
	trxsCount := []int{100, 1000}
	cache := NewTransactionCache(Config{
		MaxLen: 1100000,
	})
	for _, numTrxs := range trxsCount {
		b.Run(fmt.Sprintf("trxs count %v", numTrxs), func(b *testing.B) {
			cases := prepareTestCases(100, numTrxs)
			issuer := cases[rand.Intn(len(cases))].issuer
			for _, c := range cases {
				for _, trx := range c.transactions {
					cache.WriteIssuerSignedTransactionForReceiver(&trx)
				}
			}
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				cache.ReadAwaitingTransactionsByIssuer(issuer)
			}
		})
	}
}

func BenchmarkReadReceiver(b *testing.B) {
	trxsCount := []int{100, 1000}
	cache := NewTransactionCache(Config{
		MaxLen: 1100000,
	})
	for _, numTrxs := range trxsCount {
		b.Run(fmt.Sprintf("trxs count %v", numTrxs), func(b *testing.B) {
			cases := prepareTestCases(100, numTrxs)
			receiver := cases[rand.Intn(len(cases))].receiver
			for _, c := range cases {
				for _, trx := range c.transactions {
					cache.WriteIssuerSignedTransactionForReceiver(&trx)
				}
			}
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				cache.ReadAwaitingTransactionsByReceiver(receiver)
			}
		})
	}
}

func BenchmarkDeleteTransactions(b *testing.B) {
	trxsCount := []int{100, 1000}
	cache := NewTransactionCache(Config{
		MaxLen: 1100000,
	})
	for _, numTrxs := range trxsCount {
		b.Run(fmt.Sprintf("trxs count %v", numTrxs), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				b.StopTimer()
				cases := prepareTestCases(100, numTrxs)
				for _, c := range cases {
					for _, trx := range c.transactions {
						cache.WriteIssuerSignedTransactionForReceiver(&trx)
					}
				}
				b.StartTimer()
				for _, c := range cases {
					cache.CleanSignedTransactions(c.transactions)
				}
			}
		})
	}
}
