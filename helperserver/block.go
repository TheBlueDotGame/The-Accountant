package helperserver

import (
	"errors"

	"github.com/bartossh/Computantis/block"
)

func (a *app) validateBlock(block *block.Block) error {
	a.mux.Lock()
	defer a.mux.Unlock()
	defer func() {
		a.lastBlock = *block
	}()
	if a.lastBlock.Index != 0 {
		if block.Index != a.lastBlock.Index+1 {
			return errors.Join(ErrBlockIndexIsInvalid, errors.New("index isn't matching"))
		}
		if block.PrevHash != a.lastBlock.Hash {
			return errors.Join(ErrBlockIndexIsInvalid, errors.New("hash isn't matching"))
		}
	}
	if !block.Validate(block.TrxHashes) {
		return ErrProofBlockIsInvalid
	}
	return nil
}
