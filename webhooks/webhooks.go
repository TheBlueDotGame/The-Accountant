package webhooks

import (
	"errors"
	"fmt"
	"sync"

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/logger"
)

const (
	TriggerNewBlock       byte = iota // TriggerNewBlock is the trigger for new block. It is triggered when a new block is forged.
	TriggerNewTransaction             // TriggerNewTransaction is a trigger for new transaction. It is triggered when a new transaction is received.
)

var (
	ErrorHookNotImplemented = errors.New("hook not implemented")
)

// Hook is the hook that is used to trigger the webhook.
type Hook struct {
	URL   string `json:"address"` // URL is a url  of the webhook.
	Token string `json:"token"`   // Token is the token added to the webhook to verify that the message comes from the valid source.
}

type hooks map[string]Hook

// HookRequestHTTPPoster provides PostWebhookBlock method that allows to post new forged block to the webhook url over HTTP protocol.
type HookRequestHTTPPoster interface {
	PostWebhookBlock(url string, token string, block *block.Block) error
	PostWebhookNewTransaction(url string, token string) error
}

// Service provide webhook service that is used to create, remove and update webhooks.
type Service struct {
	mux    sync.RWMutex
	buffer map[byte]hooks
	client HookRequestHTTPPoster
	log    logger.Logger
}

// New creates new instance of the webhook service.
func New(client HookRequestHTTPPoster, l logger.Logger) *Service {
	return &Service{
		mux:    sync.RWMutex{},
		buffer: make(map[byte]hooks),
		client: client,
		log:    l,
	}
}

// CreateWebhook creates new webhook or or updates existing one for given trigger.
func (s *Service) CreateWebhook(trigger byte, publicAddress string, h Hook) error {
	switch trigger {
	case TriggerNewBlock, TriggerNewTransaction:
		s.insertHook(trigger, publicAddress, h)
	default:
		return ErrorHookNotImplemented
	}
	return nil
}

// RemoveWebhook removes webhook for given trigger and Hook URL.
func (s *Service) RemoveWebhook(trigger byte, publicAddress string, h Hook) error {
	switch trigger {
	case TriggerNewBlock, TriggerNewTransaction:
		s.removeHook(trigger, publicAddress, h)
	default:
		return ErrorHookNotImplemented
	}
	return nil
}

// PostWebhookBlock posts block to all webhooks that are subscribed to the new block trigger.
func (s *Service) PostWebhookBlock(blc *block.Block) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	hs, ok := s.buffer[TriggerNewBlock]
	if !ok {
		return
	}
	for _, h := range hs {
		if err := s.client.PostWebhookBlock(h.URL, h.Token, blc); err != nil {
			s.log.Error(fmt.Sprintf("webhook service error posting block to webhook url: %s, %s", h.URL, err.Error()))
		}
	}
}

// PostWebhookNewTransaction posts information to the coresponding public address hook url with information about new waiting transaction.
func (s *Service) PostWebhookNewTransaction(url string, token string) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	hs, ok := s.buffer[TriggerNewBlock]
	if !ok {
		return
	}
	for _, h := range hs {
		if err := s.client.PostWebhookNewTransaction(h.URL, h.Token); err != nil {
			s.log.Error(fmt.Sprintf("webhook service error posting block to webhook url: %s, %s", h.URL, err.Error()))
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
