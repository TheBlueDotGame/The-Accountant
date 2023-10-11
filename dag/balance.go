package dag

import (
	"github.com/bartossh/Computantis/spice"

	msgpackv2 "github.com/shamaton/msgpack/v2"
	"github.com/vmihailenco/msgpack"
)

// Balance holds the wallet address balance for convenience of fast lookup but it is redundant informations.
// This entity lives alongside graph as a simple directed acyclic graph itself with only one parent.
// It is not sealed by hashing or cryptographic signature, it only allows for faster accounting.
type Balance struct {
	LastTrxVectorHash [32]byte      `msgpack:"last_trx_vector_hash"`
	Spice             spice.Melange `msgpack:"spice"`
}

// NewBalance creates a new balance entity.
func NewBalance(h [32]byte, s spice.Melange) Balance {
	return Balance{LastTrxVectorHash: h, Spice: s}
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
