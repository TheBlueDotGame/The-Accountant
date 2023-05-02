package repomongo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bartossh/Computantis/logger"
)

// Write writes log to the database.
// p is a marshaled logger.Log.
func (db DataBase) Write(p []byte) (n int, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	var l logger.Log
	if err := json.Unmarshal(p, &l); err != nil {
		return 0, err
	}
	if _, err := db.inner.Collection(logsCollection).InsertOne(ctx, l); err != nil {
		return 0, err
	}
	return len(p), nil
}
