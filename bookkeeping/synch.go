package bookkeeping

import (
	"context"
	"errors"
)

var (
	ErrSynchronizerWatchFailure   = errors.New("synchronizer failure")
	ErrSynchronizerReleaseFailure = errors.New("synchronizer release failure")
	ErrSynchronizerStopped        = errors.New("synchronizer stopped")
)

type blockchainLockSubscriber interface {
	SubscribeToLockBlockchainNotification(ctx context.Context, c chan<- bool, node string)
}

type synchronizer interface {
	AddToBlockchainLockQueue(ctx context.Context, nodeID string) error
	RemoveFromBlockchainLocks(ctx context.Context, nodeID string) error
	CheckIsOnTopOfBlockchainsLocks(ctx context.Context, nodeID string) (bool, error)
}

type sync struct {
	synchro   synchronizer
	subscribe blockchainLockSubscriber
	id        string
}

// newSync creates new sync.
func newSync(id string, synchro synchronizer, subscribe blockchainLockSubscriber) sync {
	return sync{id: id, synchro: synchro, subscribe: subscribe}
}

func (s sync) waitInQueueForLock(ctx context.Context) error {
	ctxx, cancel := context.WithCancel(ctx)
	defer cancel()
	c := make(chan bool, 2)
	s.subscribe.SubscribeToLockBlockchainNotification(ctxx, c, s.id)
	if err := s.synchro.AddToBlockchainLockQueue(ctxx, s.id); err != nil {
		return errors.Join(ErrSynchronizerWatchFailure, err)
	}
	for {
		select {
		case <-ctxx.Done():
			return errors.Join(ErrSynchronizerStopped, ctxx.Err())
		case v := <-c:
			if v {
				ok, err := s.synchro.CheckIsOnTopOfBlockchainsLocks(ctx, s.id)
				if err != nil {
					return errors.Join(ErrSynchronizerWatchFailure, err)
				}
				if ok {
					return nil
				}
			}
		}
	}
}

func (s sync) releaseLock(ctx context.Context) error {
	if err := s.synchro.RemoveFromBlockchainLocks(ctx, s.id); err != nil {
		return errors.Join(ErrSynchronizerReleaseFailure, err)
	}
	return nil
}
