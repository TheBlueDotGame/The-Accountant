package logger

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Log is log marshaled and written in to the io.Writer of the helper implementing Logger abstraction.
type Log struct {
	ID        primitive.ObjectID `json:"_id"        bson:"_id"`
	Level     string             `jon:"level"       bson:"level"`
	Msg       string             `json:"msg"        bson:"msg"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}

// Logger represents an abstraction that logging helpers should implement.
type Logger interface {
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
	Fatal(msg string)
}
