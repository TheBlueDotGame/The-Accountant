package accountant

import (
	"time"

	"github.com/bartossh/Computantis/spice"

	msgpackv2 "github.com/shamaton/msgpack/v2"
	"github.com/vmihailenco/msgpack"
)

// Balance holds the  wallet balance.
type Balance struct {
	AccountedAt         time.Time     `msgpack:"accounted_at"`
	WalletPublicAddress string        `msgpack:"wallet_public_address"`
	Spice               spice.Melange `msgpack:"spice"`
}

// NewBalance creates a new balance entity.
func NewBalance(walletPubAddr string, s spice.Melange) Balance {
	now := time.Now()
	return Balance{AccountedAt: now, WalletPublicAddress: walletPubAddr, Spice: s}
}

func (b *Balance) encode() ([]byte, error) {
	buf, err := msgpack.Marshal(*b)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func decodeBalance(buf []byte) (Balance, error) {
	var b Balance
	err := msgpackv2.Unmarshal(buf, &b)
	return b, err
}
