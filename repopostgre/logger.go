package repopostgre

import (
	"encoding/json"
	"errors"

	"github.com/bartossh/Computantis/logger"
)

// Write writes log to the database.
// p is a marshaled logger.Log.
func (db DataBase) Write(p []byte) (n int, err error) {
	var l logger.Log
	if err := json.Unmarshal(p, &l); err != nil {
		return 0, errors.Join(ErrUnmarshalFailed, err)
	}
	timestamp := l.CreatedAt.UnixMicro()
	_, err = db.inner.Exec(
		"INSERT INTO logs (level, msg, created_at) VALUES ($1, $2, $3)",
		l.Level, l.Msg, timestamp,
	)
	if err != nil {
		return 0, errors.Join(ErrInsertFailed, err)
	}
	return 1, nil
}
