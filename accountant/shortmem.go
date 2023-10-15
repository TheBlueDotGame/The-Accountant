package accountant

import "sync"

type hyppocampus struct {
	mux           sync.RWMutex
	lastTwoHashes [2][32]byte
}

func (h *hyppocampus) set(hash [32]byte) {
	h.mux.Lock()
	defer h.mux.Unlock()
	if h.lastTwoHashes[0] == [32]byte{} {
		h.lastTwoHashes[0] = hash
		h.lastTwoHashes[1] = hash
		return
	}
	h.lastTwoHashes[1] = h.lastTwoHashes[0]
	h.lastTwoHashes[0] = hash
}

func (h *hyppocampus) getLast() [32]byte {
	h.mux.RLock()
	defer h.mux.RUnlock()
	return h.lastTwoHashes[0]
}

func (h *hyppocampus) getOneBeforeLast() [32]byte {
	h.mux.RLock()
	defer h.mux.RUnlock()
	return h.lastTwoHashes[1]
}
