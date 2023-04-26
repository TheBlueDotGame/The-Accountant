package validator

import "github.com/bartossh/Computantis/block"

func (a app) validateBlock(block *block.Block) error {
	if a.lastBlock.Index != 0 {
		if block.Index != a.lastBlock.Index+1 {
			return ErrBlockIndexIsInvalid
		}
		if block.PrevHash != a.lastBlock.Hash {
			return ErrBlockPrevHashIsInvalid
		}
	}
	if !block.Validate(block.TrxHashes) { // TODO: use hashes of received transactions after sending  transactions is implemented
		return ErrProofBlockIsInvalid
	}
	return nil
}
