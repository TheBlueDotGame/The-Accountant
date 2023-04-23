package logging

import (
	"encoding/json"
	"io"
	"time"

	"github.com/bartossh/Computantis/logger"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Helper helps with writing logs to io.Writers.
// Helper implements logger.Logger interface.
// Writing is done concurrently with out blocking the current thread.
type Helper struct {
	callOnErr func(error)
	writers   []io.Writer
}

// New creates new Helper.
func New(callOnErr func(error), writers ...io.Writer) Helper {
	return Helper{callOnErr: callOnErr, writers: writers}
}

// Debug writes debug log.
func (h Helper) Debug(msg string) {
	l := logger.Log{
		ID:        primitive.NewObjectID(),
		Level:     "debug",
		Msg:       msg,
		CreatedAt: time.Now(),
	}
	h.write(&l)
}

// Info writes info log.
func (h Helper) Info(msg string) {
	l := logger.Log{
		ID:        primitive.NewObjectID(),
		Level:     "info",
		Msg:       msg,
		CreatedAt: time.Now(),
	}
	h.write(&l)
}

// Warn writes warning log.
func (h Helper) Warn(msg string) {
	l := logger.Log{
		ID:        primitive.NewObjectID(),
		Level:     "warn",
		Msg:       msg,
		CreatedAt: time.Now(),
	}
	h.write(&l)
}

// Error writes error log.
func (h Helper) Error(msg string) {
	l := logger.Log{
		ID:        primitive.NewObjectID(),
		Level:     "error",
		Msg:       msg,
		CreatedAt: time.Now(),
	}
	h.write(&l)
}

// Fatal writes fatal log.
func (h Helper) Fatal(msg string) {
	l := logger.Log{
		ID:        primitive.NewObjectID(),
		Level:     "fatal",
		Msg:       msg,
		CreatedAt: time.Now(),
	}
	h.write(&l)
}

func (h Helper) write(l *logger.Log) {
	go func() {
		raw, err := json.Marshal(l)
		if err != nil {
			h.callOnErr(err)
		}
		for _, w := range h.writers {
			if _, err := w.Write(raw); err != nil {
				h.callOnErr(err)
			}
		}
	}()
}
