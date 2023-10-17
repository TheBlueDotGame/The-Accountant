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
	if vrx.Transaction.IssuerAddress == issuerAddress {
		if err := spiceOut.Supply(vrx.Transaction.Spice); err != nil {
			return errors.Join(ErrUnexpected, err)
		}
	}
	if vrx.Transaction.ReceiverAddress == issuerAddress {
		if err := spiceIn.Supply(vrx.Transaction.Spice); err != nil {
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
