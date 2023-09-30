package localcache

import (
	"errors"
	"fmt"
	"sync"

	"golang.org/x/exp/maps"

	"github.com/bartossh/Computantis/transaction"
)

var (
	ErrUnexpected               = errors.New("unexpected error")
	ErrNotAlloweReoccurringHash = errors.New("not allowed reoccurring hash")
)

type Config struct {
	MaxLen int `yaml:"max_len"`
}

// TransactionCache is designed to store data in parallel with repository.
type TransactionCache struct {
	trxs         map[[32]byte]transaction.Transaction
	issuerTrxs   map[string]map[[32]byte]struct{}
	receiverTrxs map[string]map[[32]byte]struct{}
	mux          sync.RWMutex
	maxLen       int
}

// NewTransactionCache creates a new TransactionCache according to Config.
func NewTransactionCache(cfg Config) *TransactionCache {
	if cfg.MaxLen < 1000 {
		cfg.MaxLen = 1000
	}
	return &TransactionCache{
		trxs:         make(map[[32]byte]transaction.Transaction, cfg.MaxLen),
		issuerTrxs:   make(map[string]map[[32]byte]struct{}),
		receiverTrxs: make(map[string]map[[32]byte]struct{}),
		maxLen:       cfg.MaxLen,
	}
}

// WriteIssuerSignedTransactionForReceiver writes transaction to cache if cache has enough space.
func (c *TransactionCache) WriteIssuerSignedTransactionForReceiver(trx *transaction.Transaction) error {
	c.mux.Lock()
	defer c.mux.Unlock()
	if _, ok := c.trxs[trx.Hash]; ok {
		return errors.Join(ErrNotAlloweReoccurringHash, fmt.Errorf("transaction of given hash exists [ %v ]", trx.Hash))
	}
	if len(c.trxs) == c.maxLen {
		return fmt.Errorf("cannot add to cache, max size of cache of [ %v ] has been reached", c.maxLen)
	}
	c.trxs[trx.Hash] = *trx
	issuerSet, ok := c.issuerTrxs[trx.IssuerAddress]
	if !ok {
		issuerSet = make(map[[32]byte]struct{})
	}
	issuerSet[trx.Hash] = struct{}{}
	c.issuerTrxs[trx.IssuerAddress] = issuerSet

	receiverSet, ok := c.receiverTrxs[trx.ReceiverAddress]
	if !ok {
		receiverSet = make(map[[32]byte]struct{})
	}
	receiverSet[trx.Hash] = struct{}{}
	c.receiverTrxs[trx.ReceiverAddress] = receiverSet

	return nil
}

// ReadAwaitingTransactionsByReceiver reads transaction belongint to the receiver if exists in the cache.
func (c *TransactionCache) ReadAwaitingTransactionsByReceiver(address string) ([]transaction.Transaction, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	hashes, ok := c.receiverTrxs[address]
	if !ok {
		return nil, fmt.Errorf("receiver address [ %s ] has no matching transaction", address)
	}
	trxs := make([]transaction.Transaction, 0, len(hashes))
	for hash := range hashes {
		trx, ok := c.trxs[hash]
		if !ok {
			return nil, fmt.Errorf("%w, receiver address [ %s ] exists but hash [ %v ] has no matching transaction", ErrUnexpected, address, hash)
		}
		trxs = append(trxs, trx)
	}
	return trxs, nil
}

// ReadAwaitingTransactionsByIssuer reads transaction belongint to the issuer if exists in the cache.
func (c *TransactionCache) ReadAwaitingTransactionsByIssuer(address string) ([]transaction.Transaction, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	hashes, ok := c.issuerTrxs[address]
	if !ok {
		return nil, fmt.Errorf("issuer address [ %s ] has no matching transaction", address)
	}
	trxs := make([]transaction.Transaction, 0, len(hashes))
	for hash := range hashes {
		trx, ok := c.trxs[hash]
		if !ok {
			return nil, fmt.Errorf("%w, issuer address [ %s ] exists but hash [ %v ] has no matching transaction", ErrUnexpected, address, hash)
		}
		trxs = append(trxs, trx)
	}
	return trxs, nil
}

// CleanSignedTransactions removes all the transactions with given hashes from the cache.
func (c *TransactionCache) CleanSignedTransactions(trxs []transaction.Transaction) {
	c.mux.Lock()
	defer c.mux.Unlock()
	for _, trx := range trxs {
		trx, ok := c.trxs[trx.Hash]
		if !ok {
			continue
		}
		if issuerTrxsHashes, ok := c.issuerTrxs[trx.IssuerAddress]; ok {
			maps.DeleteFunc[map[[32]byte]struct{}, [32]byte, struct{}](issuerTrxsHashes, func(k [32]byte, v struct{}) bool {
				return k == trx.Hash
			})
			if len(issuerTrxsHashes) == 0 {
				delete(c.issuerTrxs, trx.IssuerAddress)
			}
		}

		if receiverTrxsHashes, ok := c.receiverTrxs[trx.ReceiverAddress]; ok {
			maps.DeleteFunc[map[[32]byte]struct{}, [32]byte, struct{}](receiverTrxsHashes, func(k [32]byte, v struct{}) bool {
				return k == trx.Hash
			})
			if len(receiverTrxsHashes) == 0 {
				delete(c.receiverTrxs, trx.ReceiverAddress)
			}
		}
		delete(c.trxs, trx.Hash)
	}
}
