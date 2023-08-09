package zincadapter

import (
	"errors"
	"fmt"
	"time"

	"github.com/bartossh/Computantis/httpclient"
)

const (
	healthz              = "/healthz"
	createDocumentWithId = "/api/%s/_doc"
)

const timeout = time.Second * 5

var (
	ErrZincServerNotResponding = errors.New("zinc server not responding on given address")
	ErrZincServerWriteFailed   = errors.New("zinc server write failed")
)

// LoggerConfig contains configuration for logger back-end
type Config struct {
	Address string `yaml:"address"` // logger back-end server address
	Index   string `yaml:"index"`   // unique index per service to easy search for logs by the service
}

type message struct {
	AdditionalProp1 struct {
		Message string `json:"message"`
	} `json:"additionalProp1"`
}

// ZincClient provides a client that sends logs to the zincsearch backend
type ZincClient struct {
	address   string
	indexName string
}

// New creates a new ZincClient.
func New(cfg Config) (ZincClient, error) {
	if err := httpclient.MakeGet(timeout, fmt.Sprintf("%s%s", cfg.Address, healthz), nil); err != nil {
		return ZincClient{}, errors.Join(ErrZincServerNotResponding, err)
	}
	return ZincClient{cfg.Address, cfg.Index}, nil
}

// Write satisfies io.Writer abstraction.
func (z *ZincClient) Write(p []byte) (n int, err error) {
	var msg message
	msg.AdditionalProp1.Message = string(p)
	err = httpclient.MakePost(timeout, fmt.Sprintf("%s%s", z.address, fmt.Sprintf(createDocumentWithId, z.indexName)), msg, nil)
	if err != nil {
		return 0, errors.Join(ErrZincServerWriteFailed, err)
	}
	return len(p), nil
}
