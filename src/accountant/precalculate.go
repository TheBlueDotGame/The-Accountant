package accountant

import (
	"errors"

	"github.com/bartossh/Computantis/src/spice"
	msgpackv2 "github.com/shamaton/msgpack/v2"
	"github.com/vmihailenco/msgpack"
)

// Precalculatedfunds are funds of given wallet precalculated up to given hash to be saved in the storage.
type Precalculatedfunds struct {
	Spice spice.Melange `msgpack:"spice"`
}

func (p *Precalculatedfunds) encode() ([]byte, error) {
	buf, err := msgpack.Marshal(*p)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func decodePrecalulatedfunds(buf []byte) (Precalculatedfunds, error) {
	var p Precalculatedfunds
	err := msgpackv2.Unmarshal(buf, &p)
	return p, err
}

type fundsMemMap struct {
	m map[string]Precalculatedfunds
}

func newfundsMemMap() fundsMemMap {
	return fundsMemMap{m: make(map[string]Precalculatedfunds)}
}

func (f *fundsMemMap) set(address string, s *spice.Melange) {
	if s == nil {
		return
	}
	p := Precalculatedfunds{
		Spice: spice.Melange{
			Currency:              s.Currency,
			SupplementaryCurrency: s.SupplementaryCurrency,
		},
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

	f.updatefunds(vrx.Transaction.IssuerAddress, vrx.Transaction.ReceiverAddress, &vrx.Transaction.Spice)

	return nil
}

func (f *fundsMemMap) updatefunds(issuer, receiver string, s *spice.Melange) {
	ip, ok := f.m[issuer]
	if !ok {
		ip.Spice = spice.New(0, 0)
	}
	rp, ok := f.m[receiver]
	if !ok {
		rp.Spice = spice.New(0, 0)
	}

	ip.Spice.Drain(*s, &rp.Spice)

	f.m[issuer] = ip
	f.m[receiver] = rp
}

func (f *fundsMemMap) saveToStorage(savefundsToStorage func(address string, f Precalculatedfunds) error) error {
	for address, pf := range f.m {
		if err := savefundsToStorage(address, pf); err != nil {
			return err
		}
	}
	return nil
}
