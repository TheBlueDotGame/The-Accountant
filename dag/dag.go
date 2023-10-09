package dag

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/bartossh/Computantis/logger"
	badger "github.com/dgraph-io/badger/v4"
)

const (
	gcRuntimeTick = time.Minute * 5
)

const (
	keyTip                  = "tip"
	keyVerex                = "vertex"
	keyAddressLastTrxVertex = "last_addr_vrx"
)

func createKey(prefix string, key []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString(prefix)
	buf.Write(key)
	return buf.Bytes()
}

var ErrVertexRejected = errors.New("vertex rejected")

type signatureVerifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

type signer interface {
	Sign(message []byte) (digest [32]byte, signature []byte)
	Address() string
}

// AccountingBook is an entity that represents the accounting process of all received transactions.
type AccountingBook struct {
	verifier  signatureVerifier
	signer    signer
	log       logger.Logger
	db        *badger.DB
	separator []byte
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
		verifier:  verifier,
		signer:    signer,
		log:       l,
		db:        db,
		separator: []byte(cfg.Separator),
	}
	go ab.runHelper(ctx)

	return ab, nil
}

// NewVertex gets a new candidate vertex and validates it before adding to the graph.
func (ab *AccountingBook) NewVertex(vrx *Vertex) error {
	if err := vrx.verify(ab.separator, ab.verifier); err != nil {
		ab.log.Error(fmt.Sprintf("accountant [ %s ], rejected vertex [ %v ], %s", ab.signer.Address(), vrx.Hash, err))
		return errors.Join(ErrVertexRejected, err)
	}

	// TODO: check transaction transfers tokens if not, if  yes validate for double spending

	return nil
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
