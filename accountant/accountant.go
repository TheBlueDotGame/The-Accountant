package accountant

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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
	keyTip                = "tip"
	keyVertex             = "vertex"
	keyAddressBalance     = "address_balance"
	keyNodeFailureCounter = "node_failure_counter"
)

var (
	ErrLeafValidationProcessStopped       = errors.New("leaf validation process stopped")
	ErrNewLeafRejected                    = errors.New("new leaf rejected")
	ErrLeafRejected                       = errors.New("leaf rejected")
	ErrIssuerAddressBalanceNotFound       = errors.New("issuer address balance not found")
	ErrReceiverAddressBalanceNotFound     = errors.New("receiver address balance not found")
	ErrDoubleSpendingOrInsufficinetFounds = errors.New("double spending or insufficient founds")
	ErrVertexHashNotFound                 = errors.New("vertex hash not found")
	ErrUnexpected                         = errors.New("unexpected")
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
	mem           hyppocampus
	gennessisHash [32]byte
}

// New creates new AccountingBook.
func NewAccountingBook(cfg Config, verifier signatureVerifier, signer signer, l logger.Logger) (*AccountingBook, error) {
	_ = cfg // TODO: use Config when adding db
	ab := &AccountingBook{
		verifier: verifier,
		signer:   signer,
		dag:      dag.NewDAG(),
		mem:      hyppocampus{},
		log:      l,
	}

	return ab, nil
}

func (ab *AccountingBook) validateLeaf(ctx context.Context, leaf *Vertex) (bool, bool, error) {
	if leaf == nil {
		return false, false, ErrUnexpected
	}
	if err := leaf.verify(ab.verifier); err != nil {
		return false, false, errors.Join(ErrLeafRejected, err)
	}
	if !leaf.Transaction.IsSpiceTransfer() {
		_, err := ab.dag.GetVertex(string(leaf.RightParentHash[:]))
		if err != nil {
			return false, false, errors.Join(ErrLeafRejected, err)
		}

		_, err = ab.dag.GetVertex(string(leaf.LeftParentHash[:]))
		if err != nil {
			return false, false, errors.Join(ErrLeafRejected, err)
		}
		return true, true, nil
	}

	visited := make(map[string]struct{})
	spiceOut := spice.New(0, 0)
	spiceIn := spice.New(0, 0)
	var leftValid, rightValid bool
	vertices, signal, _ := ab.dag.AncestorsWalker(string(leaf.Hash[:]))

	for ancestorID := range vertices {
		select {
		case <-ctx.Done():
			signal <- true
			return false, false, ErrLeafValidationProcessStopped
		default:
		}
		if _, ok := visited[ancestorID]; ok {
			continue
		}
		visited[ancestorID] = struct{}{}

		isRoot, err := ab.dag.IsRoot(ancestorID)
		if err != nil {
			signal <- true
			return false, false, ErrUnexpected
		}
		item, err := ab.dag.GetVertex(ancestorID)
		if err != nil {
			signal <- true
			return false, false, errors.Join(ErrUnexpected, err)
		}
		switch vrx := item.(type) {
		case Vertex:
			if vrx.Hash == leaf.LeftParentHash {
				if err := vrx.verify(ab.verifier); err != nil {
					signal <- true
					return false, false, errors.Join(ErrLeafRejected, err)
				}
				leftValid = true
			}
			if vrx.Hash == leaf.RightParentHash {
				if err := vrx.verify(ab.verifier); err != nil {
					signal <- true
					return false, false, errors.Join(ErrLeafRejected, err)
				}
				rightValid = true
			}
			if err := pourFounds(leaf, &vrx, &spiceIn, &spiceOut); err != nil {
				return false, false, err
			}
			if isRoot {
				signal <- true
				return false, false, nil
			}

		default:
			signal <- true
			return false, false, ErrUnexpected
		}
	}

	if err := checkHasSufficientFounds(&spiceIn, &spiceOut); err != nil {
		return false, false, err
	}

	if !leftValid || !rightValid {
		return leftValid, rightValid, errors.Join(ErrLeafRejected, errors.New("couldn't validate left or/and right vertex"))
	}

	return leftValid, rightValid, nil
}

// CreateLeaf creates leaf vertex also known as a tip.
// Leaf validates previous leaf by:
// - checking leaf signature
// - checking leaf ancestors signatures to the depth of one parent
// - checking if leaf transferring spice issuer has enough founds.
// If leaf has valid signature and is referring to parent that is part of a graph then the leaf is valid.
// If leaf transfers spice then calculate issuer total founds and drain for given amount to calculate sufficient founds.
func (ab *AccountingBook) CreateLeaf(ctx context.Context, trx *transaction.Transaction) error {
	leavesToExamine := 2
	var err error
	validatedLeafs := make([]Vertex, 2)

	for _, item := range ab.dag.GetLeaves() {
		if leavesToExamine == 0 {
			break
		}

		var leaf Vertex
		switch vrx := item.(type) {
		case Vertex:
			leaf = vrx
			_, _, err = ab.validateLeaf(ctx, &leaf)
			if err != nil {
				ab.dag.DeleteVertex(string(leaf.Hash[:]))
				ab.log.Error(
					fmt.Sprintf("Accounting book rejected leaf hash [ %v ], from [ %v ], %s",
						leaf.Hash, leaf.SignerPublicAddress, err),
				)
				continue
			}
		default:
			return ErrUnexpected
		}

		leavesToExamine--

		validatedLeafs = append(validatedLeafs, leaf)
	}

	switch len(validatedLeafs) {
	case 2:
	case 1:
		rightHash := ab.mem.getLast()
		right, err := ab.dag.GetVertex(string(rightHash[:]))
		if err != nil {
			ab.log.Error(fmt.Sprintf("Accounting book create tip %s, %s", ErrUnexpected, err))
			return ErrUnexpected
		}
		leafRight, ok := right.(Vertex)
		if !ok {
			ab.log.Error(fmt.Sprintf("Accounting book create tip %s.", ErrUnexpected))
			return ErrUnexpected
		}
		validatedLeafs = append(validatedLeafs, leafRight)

	case 0:
		rightHash := ab.mem.getLast()
		right, err := ab.dag.GetVertex(string(rightHash[:]))
		if err != nil {
			ab.log.Error(fmt.Sprintf("Accounting book create tip %s, %s", ErrUnexpected, err))
			return ErrUnexpected
		}
		leafRight, ok := right.(Vertex)
		if !ok {
			ab.log.Error(fmt.Sprintf("Accounting book create tip %s.", ErrUnexpected))
			return ErrUnexpected
		}
		validatedLeafs = append(validatedLeafs, leafRight)

		leftHash := ab.mem.getOneBeforeLast()
		left, err := ab.dag.GetVertex(string(leftHash[:]))
		if err != nil {
			ab.log.Error(fmt.Sprintf("Accounting book create tip %s, %s", ErrUnexpected, err))
			return ErrUnexpected
		}
		leafLeft, ok := left.(Vertex)
		if !ok {
			ab.log.Error(fmt.Sprintf("Accounting book create tip %s.", ErrUnexpected))
			return ErrUnexpected
		}
		validatedLeafs = append(validatedLeafs, leafLeft)

	default:
		ab.log.Error(fmt.Sprintf("Accounting book create tip %s.", ErrUnexpected))
		return ErrUnexpected
	}

	tip, err := NewVertex(*trx, validatedLeafs[0].Hash, validatedLeafs[1].Hash, ab.signer)
	if err != nil {
		ab.log.Error(fmt.Sprintf("Accounting book rejected new leaf [ %v ], %s.", tip.Hash, err))
		return err
	}
	if err := ab.dag.AddVertexByID(string(tip.Hash[:]), tip); err != nil {
		ab.log.Error(fmt.Sprintf("Accounting book rejected new leaf [ %v ], %s.", tip.Hash, err))
		return ErrNewLeafRejected
	}

	for _, vrx := range validatedLeafs {
		if err := ab.dag.AddEdge(string(vrx.Hash[:]), string(tip.Hash[:])); err != nil {
			ab.dag.DeleteVertex(string(tip.Hash[:]))
			ab.log.Error(
				fmt.Sprintf("Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ], %s,",
					vrx.Hash, vrx.SignerPublicAddress, vrx.LeftParentHash, vrx.RightParentHash, err),
			)
			return ErrLeafRejected
		}
	}
	ab.mem.set(validatedLeafs[0].Hash)
	ab.mem.set(validatedLeafs[1].Hash)

	return nil
}

func pourFounds(leaf, vrx *Vertex, spiceIn, spiceOut *spice.Melange) error {
	if leaf == nil || vrx == nil {
		return ErrUnexpected
	}
	if spiceIn == nil || spiceOut == nil {
		return ErrUnexpected
	}
	if !vrx.Transaction.IsSpiceTransfer() {
		return nil
	}
	var sink *spice.Melange
	if vrx.Transaction.IssuerAddress == leaf.Transaction.IssuerAddress {
		sink = spiceOut
	}
	if vrx.Transaction.ReceiverAddress == leaf.Transaction.IssuerAddress {
		sink = spiceIn
	}
	if sink != nil {
		if err := vrx.Transaction.Spice.Drain(leaf.Transaction.Spice, sink); err != nil {
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

// AddLeaf adds leaf known also as tip to the graph for future validation.
// Leaf will be a subject of validation by another tip.
//func (ab *AccountingBook) AddLeaf(leaf Vertex) error {
//	if err := leaf.verify(ab.verifier); err != nil {
//		ab.log.Error(
//			fmt.Sprintf("Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ], %s.",
//				leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
//		)
//		return ErrLeafRejected
//	}
//
//	_, err = ab.dag.GetVertex(string(leaf.LeftParentHash[:]))
//}
//	if _, err := ab.dag.GetVertex(string(leaf.RightParentHash[:])); err != nil {
//		ab.log.Error(
//			fmt.Sprintf("Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ], %s.",
//				leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
//		)
//		return ErrLeafRejected
//	}
//	id, err := ab.dag.AddVertex(leaf)
//	if err != nil {
//		return ErrLeafRejected
//	}
//	for _, h := range [][32]byte{leaf.LeftParentHash, leaf.RightParentHash} {
//		if err := ab.dag.AddEdge(string(h[:]), id); err != nil {
//			ab.dag.DeleteVertex(string(leaf.Hash[:]))
//			ab.log.Error(
//				fmt.Sprintf("Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ], %s,",
//					leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
//			)
//			return ErrLeafRejected
//		}
//	}
//
//	return nil
//}
