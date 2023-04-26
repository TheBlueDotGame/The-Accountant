package webhooks

import (
	"errors"
	"fmt"
	"sync"

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/logger"
)

const (
	TriggerNewBlock = "trigger_new_block" // TriggerNewBlock is the trigger for new block. It is triggered when a new block is forged.
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

// HookRequestHTTPPoster provides PostBlock method that allows to post new forged block to the webhook url over HTTP protocol.
type HookRequestHTTPPoster interface {
	PostBlock(url string, token string, block *block.Block) error
}

// Service provide webhook service that is used to create, remove and update webhooks.
type Service struct {
	mux    sync.RWMutex
	buffer map[string]hooks
	client HookRequestHTTPPoster
	log    logger.Logger
}

// New creates new instance of the webhook service.
func New(client HookRequestHTTPPoster, l logger.Logger) *Service {
	return &Service{
		mux:    sync.RWMutex{},
		buffer: make(map[string]hooks),
		client: client,
		log:    l,
	}
}

// CreateWebhook creates new webhook.
func (s *Service) CreateWebhook(trigger string, h Hook) error {
	switch trigger {
	case TriggerNewBlock:
		s.insertHook(TriggerNewBlock, h)
	default:
		return ErrorHookNotImplemented
	}
	return nil
}

func (s *Service) insertHook(trigger string, h Hook) {
	s.mux.Lock()
	defer s.mux.Unlock()
	hs, ok := s.buffer[trigger]
	if !ok {
		hs = make(hooks)
	}
	hs[h.URL] = h
}

// RemoveWebhook removes webhook for given trigger and Hook URL.
func (s *Service) RemoveWebhook(trigger string, h Hook) error {
	switch trigger {
	case TriggerNewBlock:
		s.removeHook(TriggerNewBlock, h)
	default:
		return ErrorHookNotImplemented
	}
	return nil
}

func (s *Service) removeHook(trigger string, h Hook) {
	s.mux.Lock()
	defer s.mux.Unlock()
	hs, ok := s.buffer[trigger]
	if !ok {
		return
	}
	delete(hs, h.URL)
}

// PostBlock posts block to all webhooks that are subscribed to the new block trigger.
func (s *Service) PostBlock(blc *block.Block) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	hs, ok := s.buffer[TriggerNewBlock]
	if !ok {
		return
	}
	for _, h := range hs {
		if err := s.client.PostBlock(h.URL, h.Token, blc); err != nil {
			s.log.Error(fmt.Sprintf("webhook service error posting block to webhook url: %s, %s", h.URL, err.Error()))
		}
	}
}
