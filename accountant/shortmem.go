package accountant

import "sync"

type hyppocampus struct {
	mux          sync.RWMutex
	lastTwoHases [2][32]byte
}

func (h *hyppocampus) set(hash [32]byte) {
	h.mux.Lock()
	defer h.mux.Unlock()
	if h.lastTwoHases[0] == [32]byte{} {
		h.lastTwoHases[0] = hash
		h.lastTwoHases[1] = hash
		return
	}
	h.lastTwoHases[1] = h.lastTwoHases[0]
	h.lastTwoHases[0] = hash
}

func (h *hyppocampus) getLast() [32]byte {
	h.mux.RLock()
	defer h.mux.RUnlock()
	return h.lastTwoHases[0]
}

func (h *hyppocampus) getOneBeforeLast() [32]byte {
	h.mux.RLock()
	defer h.mux.RUnlock()
	return h.lastTwoHases[1]
}
