package helperserver

import (
	"github.com/bartossh/Computantis/block"
)

func (a *app) validateBlock(block *block.Block) error {
	a.mux.Lock()
	defer a.mux.Unlock()
	if a.lastBlock.Index != 0 {
		if block.Index != a.lastBlock.Index+1 {
			return ErrBlockIndexIsInvalid
		}
		if block.PrevHash != a.lastBlock.Hash {
			return ErrBlockPrevHashIsInvalid
		}
	}
	if !block.Validate(block.TrxHashes) {
		return ErrProofBlockIsInvalid
	}
	a.lastBlock = *block
	return nil
}
