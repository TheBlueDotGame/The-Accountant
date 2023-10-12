package accountant

import (
	"bytes"
	"errors"
	"time"

	"github.com/heimdalr/dag"

	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/spice"
	"github.com/bartossh/Computantis/transaction"
)

const (
	gcRuntimeTick = time.Minute * 5
)

const prefetchSize = 1000

const (
	leafParentsCount = 2
)

const (
	keyTip                = "tip"
	keyVertex             = "vertex"
	keyAddressBalance     = "address_balance"
	keyNodeFailureCounter = "node_failure_counter"
)

var (
	ErrVertexRejected                 = errors.New("vertex rejected")
	ErrIssuerAddressBalanceNotFound   = errors.New("issuer address balance not found")
	ErrReceiverAddressBalanceNotFound = errors.New("receiver address balance not found")
	ErrVertexHashNotFound             = errors.New("vertex hash not found")
	ErrUnexpected                     = errors.New("unexpected")
)

func createKey(prefix string, key []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString(prefix)
	buf.WriteString("-")
	buf.Write(key)
	return buf.Bytes()
}

type signatureVerifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

type signer interface {
	Sign(message []byte) (digest [32]byte, signature []byte)
	Address() string
}

// AccountingBook is an entity that represents the accounting process of all received transactions.
type AccountingBook struct {
	verifier      signatureVerifier
	signer        signer
	log           logger.Logger
	dag           *dag.DAG
	gennessisHash [32]byte
}

// New creates new AccountingBook.
func NewAccountingBook(cfg Config, verifier signatureVerifier, signer signer, l logger.Logger) (*AccountingBook, error) {
	_ = cfg // TODO: use Config when adding db
	ab := &AccountingBook{
		verifier: verifier,
		signer:   signer,
		dag:      dag.NewDAG(),
		log:      l,
	}

	return ab, nil
}

func (ab *AccountingBook) createTip(trx *transaction.Transaction) error {
	leavesToExamine := leafParentsCount
	var errM error
	validatedLeafs := make([]Vertex, 2)
Outer:
	for id, item := range ab.dag.GetLeaves() {
		if leavesToExamine > 0 {
			break
		}
		leavesToExamine--

		var leaf Vertex
		switch vrx := item.(type) {
		case Vertex:
			if err := vrx.verify(ab.verifier); err != nil {
				errM = err
				break
			}
			leaf = vrx
		default:
			errM = ErrUnexpected
			break Outer
		}

		visited := make(map[string]struct{})
		spiceOut := spice.New(0, 0)
		spiceIn := spice.New(0, 0)
		vertices, signal, _ := ab.dag.AncestorsWalker(id)
	Inner:
		for ancestorID := range vertices {
			if _, ok := visited[ancestorID]; ok {
				continue Inner
			}
			visited[ancestorID] = struct{}{}

			ok, err := ab.dag.IsRoot(ancestorID)
			if ok {
				signal <- true
				break Inner
			}
			if err != nil {
				signal <- true
				errM = err
				break Outer
			}
			item, err := ab.dag.GetVertex(ancestorID)
			if err != nil {
				signal <- true
				errM = err
				break Outer
			}
			switch vrx := item.(type) {
			case Vertex:
				var sink *spice.Melange
				if vrx.Transaction.IssuerAddress == leaf.Transaction.IssuerAddress {
					sink = &spiceOut
				}
				if vrx.Transaction.ReceiverAddress == leaf.Transaction.IssuerAddress {
					sink = &spiceIn
				}
				if sink != nil {
					if err := vrx.Transaction.Spice.Drain(leaf.Transaction.Spice, sink); err != nil {
						signal <- true
						errM = err
						break Outer
					}
				}
			default:
				signal <- true
				errM = ErrUnexpected
				break Outer
			}
		}
		sink := spice.New(0, 0)
		if err := spiceIn.Drain(spiceOut, &sink); err != nil {
			errM = err
			break Outer
		}

		validatedLeafs = append(validatedLeafs, leaf)
	}

	if errM != nil {
		return errM
	}
	if len(validatedLeafs) != 2 {
		return ErrUnexpected
	}

	tip, err := NewVertex(*trx, validatedLeafs[0].Hash, validatedLeafs[1].Hash, ab.signer)
	if err != nil {
		return err
	}
	if err := ab.dag.AddVertexByID(string(tip.Hash[:]), tip); err != nil {
		return err
	}

	for _, vrx := range validatedLeafs {
		if err := ab.dag.AddEdge(string(vrx.Hash[:]), string(tip.Hash[:])); err != nil {
			ab.dag.DeleteVertex(string(tip.Hash[:]))
			return err
		}
	}

	return nil
}
