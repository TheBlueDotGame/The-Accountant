package accountant

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bartossh/Computantis/logger"
	"github.com/dgraph-io/badger/v4"
)

func createBadgerDB(ctx context.Context, path string, l logger.Logger) (*badger.DB, error) {
	var opt badger.Options
	switch path {
	case "":
		opt = badger.DefaultOptions("").WithInMemory(true)
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
