package logger

import (
	"time"
)

// Log is log marshaled and written in to the io.Writer of the helper implementing Logger abstraction.
type Log struct {
	ID        any       `json:"_id"        bson:"_id"        db:"id"`
	CreatedAt time.Time `json:"created_at" bson:"created_at" db:"created_at"`
	Level     string    `jon:"level"       bson:"level"      db:"level"`
	Msg       string    `json:"msg"        bson:"msg"        db:"msg"`
}

// Logger provides logging methods for debug, info, warning, error and fatal.
type Logger interface {
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
	Fatal(msg string)
}
