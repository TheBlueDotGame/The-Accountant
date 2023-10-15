package accountant

import (
	"errors"

	"github.com/bartossh/Computantis/spice"
)

func pourFounds(issuerAddress string, vrx Vertex, spiceIn, spiceOut *spice.Melange) error {
	if spiceIn == nil || spiceOut == nil {
		return ErrUnexpected
	}
	if !vrx.Transaction.IsSpiceTransfer() {
		return nil
	}
	var sink *spice.Melange
	if vrx.Transaction.IssuerAddress == issuerAddress {
		sink = spiceOut
	}
	if vrx.Transaction.ReceiverAddress == issuerAddress {
		sink = spiceIn
	}
	if sink != nil {
		if err := vrx.Transaction.Spice.Drain(vrx.Transaction.Spice, sink); err != nil {
			return errors.Join(ErrUnexpected, err)
		}
	}
	return nil
}

func checkHasSufficientFounds(in, out *spice.Melange) error {
	if in == nil || out == nil {
		return ErrUnexpected
	}
	sink := spice.New(0, 0)
	if err := in.Drain(*out, &sink); err != nil {
		return errors.Join(ErrLeafRejected, err)
	}
	return nil
}
