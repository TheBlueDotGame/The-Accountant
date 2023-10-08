package dag

import (
	"fmt"
	"os"

	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/transaction"
	badger "github.com/dgraph-io/badger/v4"
)

type Vertex struct {
	Transaction     transaction.Transaction `msgpack:"transaction"`
	Hash            [32]byte                `msgpack:"hash"`
	LeftParentHash  [32]byte                `msgpack:"left_parent_hash"`
	RightParentHash [32]byte                `msgpack:"right_parent_hash"`
}

// Config contains configuration for the AccountingBook.
type Config struct {
	DBPath string `yaml:"db_path"`
}

// AccountingBook is an entity that represents the accounting process of all received transactions.
type AccountingBook struct {
	db  *badger.DB
	log logger.Logger
}

// New creates new AccountingBook.
func New(cfg Config, l logger.Logger) (*AccountingBook, error) {
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

	return &AccountingBook{
		db:  db,
		log: l,
	}, nil
}
