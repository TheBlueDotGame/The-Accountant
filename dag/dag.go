package dag

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/transaction"
	badger "github.com/dgraph-io/badger/v4"
)

const (
	gcRuntimeTick = time.Minute * 5
)

const prefetchSize = 1000

const depth = 2

const (
	keyTip                = "tip"
	keyVertex             = "vertex"
	keyAddressBalance     = "address_balance"
	keyNodeFailureCounter = "node_failure_counter"
)

var (
	ErrVertexRejected                 = errors.New("vertex rejected")
	ErrIssuerAddressBalanceNotFound   = errors.New("issuer address balance not found")
	ErrReceiverAddressBalanceNotFound = errors.New("receiver address balance not found")
	ErrVertexHashNotFound             = errors.New("vertex hash not found")
)

func createKey(prefix string, key []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString(prefix)
	buf.WriteString("-")
	buf.Write(key)
	return buf.Bytes()
}

type signatureVerifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

type signer interface {
	Sign(message []byte) (digest [32]byte, signature []byte)
	Address() string
}

// AccountingBook is an entity that represents the accounting process of all received transactions.
type AccountingBook struct {
	verifier      signatureVerifier
	signer        signer
	log           logger.Logger
	db            *badger.DB
	hippo         hippocampus
	gennessisHash [32]byte
}

// New creates new AccountingBook.
func NewAccountingBook(ctx context.Context, cfg Config, verifier signatureVerifier, signer signer, l logger.Logger) (*AccountingBook, error) {
	var opt badger.Options
	switch cfg.DBPath {
	case "":
		l.Warn("Accounting Book runs in ephemeral memory mode")
		opt = badger.DefaultOptions("").WithInMemory(true)
	default:
		if _, err := os.Stat(cfg.DBPath); err != nil {
			l.Warn(fmt.Sprintf("Accounting Book creates persistent database in file: %s", cfg.DBPath))
		}
		l.Warn(fmt.Sprintf("Accounting Book will write to persistent file: %s", cfg.DBPath))
		opt = badger.DefaultOptions(cfg.DBPath)
	}

	db, err := badger.Open(opt)
	if err != nil {
		return nil, err
	}
	ab := &AccountingBook{
		verifier: verifier,
		signer:   signer,
		log:      l,
		db:       db,
		hippo:    hippocampus{},
	}
	go ab.runHelper(ctx)

	return ab, nil
}

// CreateNewTip creates tip that awaits to be validated and added as a Vertex.
func (ab *AccountingBook) CreateNewTip(trx *transaction.Transaction, nodeID string) error {
	switch trx.IsContract() {
	case true:
		if err := trx.VerifyIssuerReceiver(ab.verifier); err != nil {
			return err
		}
	default:
		if err := trx.VerifyIssuer(ab.verifier); err != nil {
			return err
		}
	}

	blcIssuer, blcReceiver, err := ab.precalculateTransferFounds(trx)
	if err != nil {
		switch {
		case errors.Is(err, ErrIssuerAddressBalanceNotFound):
			// TODO: act on issuer balance not found
		case errors.Is(err, ErrReceiverAddressBalanceNotFound):
			// TODO: act on receiver balance not found
		default:
			ab.log.Error(fmt.Sprintf("Accounting book spice transfer in trx: [ %+v ], err: [ %s ]", trx, err))
			return err
		}
	}
	_ = blcIssuer
	_ = blcReceiver

	tipLeft, tipRight, err := ab.getTipsToValidate()
	if err != nil {
		ab.log.Error(fmt.Sprintf("Accounting book cannot get tips to validate, %s", err))
		return err
	}
	if err := ab.validateTipsOnGraph(tipLeft, tipRight); err != nil {
		ab.log.Error(fmt.Sprintf("Accounting book rejected tip validation, %s", err))
		// TODO: act on invalid graph
		return err
	}

	// TODO: update graph, balance and hippocampus,

	return nil
}

func (ab *AccountingBook) precalculateTransferFounds(trx *transaction.Transaction) (blcIssuer Balance, blcReceiver Balance, err error) {
	if err = ab.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(createKey(keyAddressBalance, []byte(trx.IssuerAddress)))
		if err != nil {
			switch err {
			case badger.ErrKeyNotFound:
				return ErrIssuerAddressBalanceNotFound
			default:
				return err
			}
		}
		if err = item.Value(func(val []byte) error {
			var err error
			blcIssuer, err = decodeBalance(val)
			return err
		}); err != nil {
			return err
		}

		item, err = txn.Get(createKey(keyAddressBalance, []byte(trx.ReceiverAddress)))
		if err != nil {
			switch err {
			case badger.ErrKeyNotFound:
				return ErrReceiverAddressBalanceNotFound
			default:
				return err
			}
		}
		if err = item.Value(func(val []byte) error {
			var err error
			blcReceiver, err = decodeBalance(val)
			return err
		}); err != nil {
			return err
		}

		return err
	}); err != nil {
		return
	}

	if trx.IsSpiceTransfer() {
		if err = blcIssuer.Spice.Drain(trx.Spice, &blcReceiver.Spice); err != nil {
			return
		}
	}

	return
}

func (ab *AccountingBook) getTipsToValidate() (tipLeft Vertex, tipRight Vertex, err error) {
	err = ab.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 1000
		it := txn.NewIterator(opts)
		var firstChecked bool
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			var tip Vertex
			if err := item.Value(func(v []byte) error {
				var err error
				tip, err = decodeVertex(v)
				return err
			}); err != nil {
				return err
			}

			if !firstChecked {
				firstChecked = true
				tipLeft = tip
				continue
			}

			if tip.Weight < tipLeft.Weight {
				tipRight = tipLeft
				tipLeft = tip
			}
		}

		if tipLeft.Weight == 0 {
			hashLeft, hashRight := ab.hippo.remindLastAndOneBeforeLastVertexHash()
			item, err := txn.Get(createKey(keyVertex, hashLeft[:]))
			if err != nil {
				return err
			}
			if err := item.Value(func(val []byte) error {
				var err error
				tipLeft, err = decodeVertex(val)
				return err
			}); err != nil {
				return err
			}
			item, err = txn.Get(createKey(keyVertex, hashRight[:]))
			if err != nil {
				return err
			}
			if err := item.Value(func(val []byte) error {
				var err error
				tipRight, err = decodeVertex(val)
				return err
			}); err != nil {
				return err
			}
		}
		if tipRight.Weight == 0 {
			_, hashRight := ab.hippo.remindLastAndOneBeforeLastVertexHash()
			item, err := txn.Get(createKey(keyVertex, hashRight[:]))
			if err != nil {
				return err
			}
			if err := item.Value(func(val []byte) error {
				var err error
				tipRight, err = decodeVertex(val)
				return err
			}); err != nil {
				return err
			}
		}
		return nil
	})
	return
}

func (ab *AccountingBook) validateTipsOnGraph(tipLeft, tipRight Vertex) error {
	return ab.db.View(func(txn *badger.Txn) error {
		for _, current := range []Vertex{tipLeft, tipRight} {
			d := depth
			for d > 0 {
				h := current.RightParentHash
				if rand.Intn(2) == 1 {
					h = current.LeftParentHash
				}
				if h == ab.gennessisHash {
					return nil
				}
				item, err := txn.Get(createKey(keyVertex, h[:]))
				if err != nil {
					switch {
					case errors.Is(err, badger.ErrKeyNotFound):
						return ErrVertexHashNotFound
					default:
						return err
					}
				}
				if err := item.Value(func(val []byte) error {
					var err error
					current, err = decodeVertex(val)
					return err
				}); err != nil {
					return err
				}
				d--
			}
		}
		return nil
	})
}

func (ab *AccountingBook) runHelper(ctx context.Context) {
	ticker := time.NewTicker(gcRuntimeTick)
	defer ab.db.Close()
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
		again:
			err := ab.db.RunValueLogGC(0.5)
			if err == nil {
				goto again
			}
		case <-ctx.Done():
			return
		}
	}
}
