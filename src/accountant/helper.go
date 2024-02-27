package accountant

import "errors"

type strBufer struct {
	pos   int
	inner []string
}

func newStrBufer(cap int) strBufer {
	return strBufer{0, make([]string, 0, cap)}
}

func (b *strBufer) add(s string) {
	b.pos = 0
	b.inner = append(b.inner, s)
}

func (b *strBufer) next() (string, bool) {
	if b.pos == len(b.inner) {
		b.pos = 0
		return "", false
	}
	result := b.inner[b.pos]
	b.pos++
	return result, true
}

type hashAtDepth struct {
	hash  [32]byte
	depth uint64
}

func newHashAtDepth(depth uint64) hashAtDepth {
	return hashAtDepth{depth: depth}
}

func (h *hashAtDepth) next(v *Vertex) error {
	if v == nil {
		return errors.New("vertex cannot be nil")
	}
	h.depth--
	if h.depth == 0 {
		h.hash = v.Hash
		return ErrBreak
	}
	return nil
}

func (h *hashAtDepth) getHash() [32]byte {
	return h.hash
}
