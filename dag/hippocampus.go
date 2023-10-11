package dag

import "sync"

type hippocampus struct {
	mux                     sync.RWMutex
	lastVertexHash          [32]byte
	oneBeforeLastVertexHash [32]byte
}

func (h *hippocampus) rememberLastVertexHash(vrxHash [32]byte) {
	h.mux.Lock()
	defer h.mux.Unlock()
	switch h.lastVertexHash {
	case [32]byte{}:
		h.oneBeforeLastVertexHash = vrxHash
	default:
		h.oneBeforeLastVertexHash = h.lastVertexHash
	}
	h.lastVertexHash = vrxHash
}

func (h *hippocampus) remindLastAndOneBeforeLastVertexHash() ([32]byte, [32]byte) {
	h.mux.RLock()
	defer h.mux.RUnlock()
	return h.lastVertexHash, h.oneBeforeLastVertexHash
}
