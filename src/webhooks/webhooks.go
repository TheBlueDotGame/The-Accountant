package webhooks

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bartossh/Computantis/src/httpclient"
	"github.com/bartossh/Computantis/src/logger"
)

const (
	TriggerNewTransaction = iota // TriggerNewTransaction is a trigger for new transaction. It is triggered when a new transaction is received.
)

var ErrorHookNotImplemented = errors.New("hook not implemented")

const (
	StateIssued      byte = 0 // StateIssued is state of the transaction meaning it is only signed by the issuer.
	StateAcknowleged          // StateAcknowledged is a state ot the transaction meaning it is acknowledged and signed by the receiver.
)

// AwaitingTransactionsMessage is the message send to the webhook url about new transaction for given wallet address.
type AwaitingTransactionsMessage struct {
	Time          time.Time `json:"time"`
	NotaryNodeURL string    `json:"notary_node_url"` // NotaryNodeURL tells the webhook creator where the transactions which notary node stores transactions.
	State         byte      `json:"state"`
}

// Hook is the hook that is used to trigger the webhook.
type Hook struct {
	URL string `json:"address"` // URL is a url  of the webhook.
}

type hooks map[string]Hook

// Service provide webhook service that is used to create, remove and update webhooks.
type Service struct {
	buffer map[byte]hooks
	log    logger.Logger
	mux    sync.RWMutex
}

// New creates new instance of the webhook service.
func New(l logger.Logger) *Service {
	return &Service{
		mux:    sync.RWMutex{},
		buffer: make(map[byte]hooks),
		log:    l,
	}
}

// CreateWebhook creates new webhook or or updates existing one for given trigger.
func (s *Service) CreateWebhook(trigger byte, publicAddress string, h Hook) error {
	switch trigger {
	case TriggerNewTransaction:
		s.insertHook(trigger, publicAddress, h)
	default:
		return ErrorHookNotImplemented
	}
	return nil
}

// RemoveWebhook removes webhook for given trigger and Hook URL.
func (s *Service) RemoveWebhook(trigger byte, publicAddress string, h Hook) error {
	switch trigger {
	case TriggerNewTransaction:
		s.removeHook(trigger, publicAddress, h)
	default:
		return ErrorHookNotImplemented
	}
	return nil
}

// PostWebhookNewTransaction posts information to the corresponding public address.
func (s *Service) PostWebhookNewTransaction(publicAddresses []string, storingNodeURL string) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	hs, ok := s.buffer[TriggerNewTransaction]
	if !ok {
		return
	}

	in := make(map[string]interface{})
	for _, addr := range publicAddresses {
		h, ok := hs[addr]
		if !ok {
			continue
		}

		transactionMsg := AwaitingTransactionsMessage{
			NotaryNodeURL: storingNodeURL,
			State:         StateIssued,
			Time:          time.Now(),
		}
		if err := httpclient.MakePost(time.Second*5, h.URL, transactionMsg, &in); err != nil {
			s.log.Error(fmt.Sprintf("webhook service error posting transaction to webhook URL: %s, %s", h.URL, err.Error()))
		}
	}
}

func (s *Service) insertHook(trigger byte, publicAddress string, h Hook) {
	s.mux.Lock()
	defer s.mux.Unlock()
	hs, ok := s.buffer[trigger]
	if !ok {
		hs = make(hooks)
	}
	hs[publicAddress] = h
	s.buffer[trigger] = hs
}

func (s *Service) removeHook(trigger byte, publicAddress string, h Hook) {
	s.mux.Lock()
	defer s.mux.Unlock()
	hs, ok := s.buffer[trigger]
	if !ok {
		return
	}
	delete(hs, publicAddress)
}
