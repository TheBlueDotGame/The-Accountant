package accountant

import (
	"errors"

	"github.com/bartossh/Computantis/src/spice"
)

// Precalculatedfunds are funds of given wallet precalculated up to given hash to be saved in the storage.
type precalculatedFounds struct {
	in  spice.Melange
	out spice.Melange
}

type fundsMemMap struct {
	m map[string]precalculatedFounds
}

func newFoundsMemMap() fundsMemMap {
	return fundsMemMap{m: make(map[string]precalculatedFounds)}
}

func (f *fundsMemMap) set(address string, s *spice.Melange) {
	if s == nil {
		return
	}
	p := precalculatedFounds{
		in: s.Clone(),
	}
	f.m[address] = p
}

func (f *fundsMemMap) nextVertex(vrx *Vertex) error {
	if vrx == nil {
		return errors.Join(ErrUnexpected, errors.New("next vertex cannot be nil"))
	}

	if !vrx.Transaction.IsSpiceTransfer() {
		return nil
	}

	f.updateFounds(vrx.Transaction.IssuerAddress, vrx.Transaction.ReceiverAddress, &vrx.Transaction.Spice)

	return nil
}

func (f *fundsMemMap) updateFounds(issuer, receiver string, s *spice.Melange) {
	ip := f.m[issuer]
	rp := f.m[receiver]
	ip.out.Supply(*s)
	rp.in.Supply(*s)
	f.m[issuer] = ip
	f.m[receiver] = rp
}

func (f *fundsMemMap) saveToStorage(saveFoundsToStorage func(address string, s spice.Melange) error) error {
	for address, pf := range f.m {
		pf.in.Drain(pf.out, &spice.Melange{})
		if err := saveFoundsToStorage(address, pf.in); err != nil {
			return err
		}
	}
	return nil
}
