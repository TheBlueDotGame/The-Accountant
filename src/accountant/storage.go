package accountant

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/bartossh/Computantis/src/logger"
	"github.com/bartossh/Computantis/src/transaction"
	"github.com/dgraph-io/badger/v4"
)

const (
	gcRuntimeTick = time.Minute * 5
)

func createBadgerDB(ctx context.Context, path string, l logger.Logger, detectConflicts bool) (*badger.DB, error) {
	var opt badger.Options
	switch path {
	case "":
		opt = badger.DefaultOptions("").WithInMemory(true).WithDetectConflicts(detectConflicts)
	default:
		if _, err := os.Stat(path); err != nil {
			return nil, err
		}
		opt = badger.DefaultOptions(path)
	}

	db, err := badger.Open(opt)
	if err != nil {
		return nil, err
	}

	go func(ctx context.Context) {
		ticker := time.NewTicker(gcRuntimeTick)
		defer ticker.Stop()
		for range ticker.C {
			select {
			case <-ctx.Done():
				db.Close()
				return
			default:
			}
			err := db.RunValueLogGC(0.5)
			if err == nil {
				l.Debug(fmt.Sprintf("badger DB garbage collection loop failure: %s", err))
			}
		}
	}(ctx)

	return db, nil
}

func (ab *AccountingBook) checkIsTrustedNode(trustedNodePublicAddress string) (bool, error) {
	var ok bool
	err := ab.trustedNodesDB.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(trustedNodePublicAddress))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return nil
			}
			return err
		}
		ok = true
		return nil
	})
	return ok, err
}

func (ab *AccountingBook) checkTrxInVertexExists(trxHash []byte) (bool, error) {
	err := ab.trxsToVertxDB.View(func(txn *badger.Txn) error {
		_, err := txn.Get(trxHash)
		if err != nil {
			return err
		}
		return nil
	})
	if err == nil {
		return true, nil
	}
	switch err {
	case badger.ErrKeyNotFound:
		return false, nil
	default:
		ab.log.Error(fmt.Sprintf("transaction to vertex mapping for existing trx lookup failed, %s", err))
		return false, ErrUnexpected
	}
}

func (ab *AccountingBook) saveTrxInVertex(trxHash, vrxHash []byte) error {
	return ab.trxsToVertxDB.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(trxHash); err == nil {
			return ErrTrxInVertexAlreadyExists
		}
		return txn.SetEntry(badger.NewEntry(trxHash, vrxHash))
	})
}

func (ab *AccountingBook) removeTrxInVertex(trxHash []byte) error {
	return ab.trxsToVertxDB.Update(func(txn *badger.Txn) error {
		return txn.Delete(trxHash)
	})
}

func (ab *AccountingBook) readTrxVertex(trxHash []byte) (Vertex, error) {
	var vrxHash []byte
	err := ab.trxsToVertxDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get(trxHash)
		if err != nil {
			return err
		}
		item.Value(func(v []byte) error {
			vrxHash = v
			return nil
		})
		return nil
	})
	if err != nil {
		switch err {
		case badger.ErrKeyNotFound:
			return Vertex{}, ErrTrxToVertexNotFound
		default:
			ab.log.Error(fmt.Sprintf("transaction to vertex mapping failed when looking for transaction hash, %s", err))
			return Vertex{}, ErrUnexpected
		}
	}
	return ab.readVertex(vrxHash)
}

func (ab *AccountingBook) readVertexFromStorage(vrxHash []byte) (Vertex, error) {
	var vrx Vertex
	err := ab.verticesDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get(vrxHash)
		if err != nil {
			return err
		}
		item.Value(func(v []byte) error {
			vrx, err = decodeVertex(v)
			return err
		})
		return nil
	})
	if err != nil {
		switch err {
		case badger.ErrKeyNotFound:
			return vrx, ErrVertexHashNotFound
		default:
			ab.log.Error(fmt.Sprintf("transaction to vertex mapping failed when looking for vertex hash, %s", err))
			return vrx, ErrUnexpected
		}
	}

	return vrx, nil
}

func (ab *AccountingBook) saveVertexToStorage(vrx *Vertex) error {
	buf, err := vrx.Encode()
	if err != nil {
		return err
	}
	return ab.verticesDB.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(vrx.Hash[:]); err == nil {
			return ErrVertexAlreadyExists
		}
		return txn.SetEntry(badger.NewEntry(vrx.Hash[:], buf))
	})
}

func (ab *AccountingBook) readVertexHashContainingTrxHashFromStorage(hash [32]byte) ([]byte, error) {
	var vertexHash []byte
	if err := ab.trxsToVertxDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get(hash[:])
		if err != nil {
			if !errors.Is(err, badger.ErrKeyNotFound) {
				ab.log.Error(fmt.Sprintf("accountant error with reading transaction to vertex mapping, %s", err))
			}
			return err
		}

		item.Value(func(val []byte) error {
			vertexHash = val
			return nil
		})
		return nil
	}); err != nil {
		return []byte{}, err
	}
	return vertexHash, nil
}

func (ab *AccountingBook) readTransactionFromStorage(vertexHash []byte) (transaction.Transaction, error) {
	var trx transaction.Transaction
	if err := ab.verticesDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get(vertexHash)
		if err != nil {
			if !errors.Is(err, badger.ErrKeyNotFound) {
				ab.log.Error(fmt.Sprintf("accountant error with reading vertex from DB, %s", err))
				return err
			}
			return ErrEntityNotFound
		}

		item.Value(func(val []byte) error {
			vrx, err := decodeVertex(val)
			if err != nil {
				return errors.Join(ErrUnexpected, err)
			}
			trx = vrx.Transaction
			return nil
		})
		return nil
	}); err != nil {
		return trx, err
	}

	return trx, nil
}

func (ab *AccountingBook) checkVertexExistInStorage(vrxHash []byte) (bool, error) {
	err := ab.verticesDB.View(func(txn *badger.Txn) error {
		_, err := txn.Get(vrxHash)
		return err
	})
	if err != nil {
		switch err {
		case badger.ErrKeyNotFound:
			return false, nil
		default:
			ab.log.Error(fmt.Sprintf("transaction to vertex mapping failed when looking for transaction hash, %s", err))
			return false, ErrUnexpected
		}
	}
	return true, nil
}
