package storage

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bartossh/Computantis/src/logger"
	"github.com/dgraph-io/badger/v4"
)

const (
	gcRuntimeTick = time.Minute * 5
)

// Create storage returns a BadgerDB storage and runs the Garbage Collection concurrently.
// To stop the storage and disconnect from database cancel the context.
func CreateBadgerDB(ctx context.Context, path string, l logger.Logger, detectConflicts bool) (*badger.DB, error) {
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
