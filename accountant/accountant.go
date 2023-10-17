package accountant

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/dgraph-io/badger/v4"
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
	keyAllowdWalletsPubAddress = "key_allowed_wallets_pub_address"
	keyTokens                  = "key_tokens"
	keyNotaryNodesPubAddress   = "key_notary_node_pub_address"
)

var (
	ErrGenesisRejected                       = errors.New("genesis vertex has been rejected")
	ErrBalanceCaclulationUnexpectedFailure   = errors.New("balance calculation unexpected failure")
	ErrBalanceUnavailable                    = errors.New("balance unavailable")
	ErrLeafBallanceCalculationProcessStopped = errors.New("wallet balance calculation process stopped")
	ErrLeafValidationProcessStopped          = errors.New("leaf validation process stopped")
	ErrNewLeafRejected                       = errors.New("new leaf rejected")
	ErrLeafRejected                          = errors.New("leaf rejected")
	ErrIssuerAddressBalanceNotFound          = errors.New("issuer address balance not found")
	ErrReceiverAddressBalanceNotFound        = errors.New("receiver address balance not found")
	ErrDoubleSpendingOrInsufficinetFounds    = errors.New("double spending or insufficient founds")
	ErrVertexHashNotFound                    = errors.New("vertex hash not found")
	ErrUnexpected                            = errors.New("unexpected failure")
	ErrTransferringFoundsFailure             = errors.New("transferring founds failure")
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
	verifier       signatureVerifier
	signer         signer
	log            logger.Logger
	dag            *dag.DAG
	db             *badger.DB
	lastVertexHash chan [32]byte
	registry       chan struct{}
	gennessisHash  [32]byte
}

// New creates new AccountingBook.
// New AccountingBook will start internally the garbage collection loop, to stop it from running cancel the context.
func NewAccountingBook(ctx context.Context, cfg Config, verifier signatureVerifier, signer signer, l logger.Logger) (*AccountingBook, error) {
	var opt badger.Options
	switch cfg.DBPath {
	case "":
		opt = badger.DefaultOptions("").WithInMemory(true)
	default:
		if _, err := os.Stat(cfg.DBPath); err != nil {
			return nil, err
		}
		opt = badger.DefaultOptions(cfg.DBPath)
	}

	db, err := badger.Open(opt)
	if err != nil {
		return nil, err
	}
	ab := &AccountingBook{
		verifier:       verifier,
		signer:         signer,
		dag:            dag.NewDAG(),
		db:             db,
		lastVertexHash: make(chan [32]byte, 100),
		registry:       make(chan struct{}, 1),
		log:            l,
	}

	go func(ctx context.Context) {
		ticker := time.NewTicker(gcRuntimeTick)
		defer ticker.Stop()
		for range ticker.C {
			select {
			case <-ctx.Done():
				return
			default:
			}
		again:
			err := db.RunValueLogGC(0.5)
			if err == nil {
				goto again
			}
		}
	}(ctx)

	ab.unregister()

	return ab, nil
}

func (ab *AccountingBook) validateLeaf(ctx context.Context, leaf *Vertex) error {
	if leaf == nil {
		return errors.Join(ErrUnexpected, errors.New("leaf to validate is nil"))
	}
	if err := leaf.verify(ab.verifier); err != nil {
		return errors.Join(ErrLeafRejected, err)
	}
	isRoot, err := ab.dag.IsRoot(string(leaf.Hash[:]))
	if err != nil {
		return errors.Join(ErrUnexpected, err)
	}
	if isRoot {
		return nil
	}
	trusted, err := ab.checkIsTrustedNode(leaf.SignerPublicAddress)
	if err != nil {
		return errors.Join(ErrUnexpected, err)
	}
	if !leaf.Transaction.IsSpiceTransfer() || trusted {
		_, err := ab.dag.GetVertex(string(leaf.RightParentHash[:]))
		if err != nil {
			return errors.Join(ErrLeafRejected, err)
		}

		_, err = ab.dag.GetVertex(string(leaf.LeftParentHash[:]))
		if err != nil {
			return errors.Join(ErrLeafRejected, err)
		}
		return nil
	}

	visited := make(map[string]struct{})
	spiceOut := spice.New(0, 0)
	spiceIn := spice.New(0, 0)
	if err := pourFounds(leaf.Transaction.IssuerAddress, *leaf, &spiceIn, &spiceOut); err != nil {
		return err
	}
	vertices, signal, _ := ab.dag.AncestorsWalker(string(leaf.Hash[:]))
	for ancestorID := range vertices {
		select {
		case <-ctx.Done():
			signal <- true
			return ErrLeafValidationProcessStopped
		default:
		}
		if _, ok := visited[ancestorID]; ok {
			continue
		}
		visited[ancestorID] = struct{}{}

		item, err := ab.dag.GetVertex(ancestorID)
		if err != nil {
			signal <- true
			return errors.Join(ErrUnexpected, err)
		}
		switch vrx := item.(type) {
		case *Vertex:
			if vrx == nil {
				return ErrUnexpected
			}
			if vrx.Hash == leaf.LeftParentHash {
				if err := vrx.verify(ab.verifier); err != nil {
					signal <- true
					return errors.Join(ErrLeafRejected, err)
				}
			}
			if vrx.Hash == leaf.RightParentHash {
				if err := vrx.verify(ab.verifier); err != nil {
					signal <- true
					return errors.Join(ErrLeafRejected, err)
				}
			}
			if err := pourFounds(leaf.Transaction.IssuerAddress, *vrx, &spiceIn, &spiceOut); err != nil {
				return errors.Join(ErrTransferringFoundsFailure, err)
			}

		default:
			signal <- true
			return ErrUnexpected
		}
	}

	err = checkHasSufficientFounds(&spiceIn, &spiceOut)
	if err != nil {
		return errors.Join(ErrTransferringFoundsFailure, err)
	}
	return nil
}

func (ab *AccountingBook) checkIsTrustedNode(trustedNodePublicAddress string) (bool, error) {
	var ok bool
	err := ab.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(createKey(keyNotaryNodesPubAddress, []byte(trustedNodePublicAddress)))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return nil
			}
			return err
		}
		ok = true
		return nil
	})
	return ok, err
}

func (ab *AccountingBook) register() {
	<-ab.registry
}

func (ab *AccountingBook) unregister() {
	ab.registry <- struct{}{}
}

// CreateGenesis creates genesis vertex that will transfer spice to current node as a receiver.
func (ab *AccountingBook) CreateGenesis(subject string, spc spice.Melange, data []byte, receiver signer) (Vertex, error) {
	ab.register()
	defer ab.unregister()
	trx, err := transaction.New(subject, spc, data, receiver.Address(), ab.signer)
	if err != nil {
		return Vertex{}, errors.Join(ErrGenesisRejected, err)
	}

	vrx, err := NewVertex(trx, [32]byte{}, [32]byte{}, ab.signer)
	if err != nil {
		return Vertex{}, errors.Join(ErrGenesisRejected, err)
	}

	if err := ab.dag.AddVertexByID(string(vrx.Hash[:]), &vrx); err != nil {
		return Vertex{}, err
	}
	ab.lastVertexHash <- vrx.Hash
	ab.lastVertexHash <- vrx.Hash

	return vrx, nil
}

// AddTrustedNode adds trusted node public address to the trusted nodes public address repository.
func (ab *AccountingBook) AddTrustedNode(trustedNodePublicAddress string) error {
	return ab.db.Update(func(txn *badger.Txn) error {
		return txn.Set(createKey(keyNotaryNodesPubAddress, []byte(trustedNodePublicAddress)), []byte{})
	})
}

// RemoveTrustedNode removes trusted node public address from trusted nodes public address repository.
func (ab *AccountingBook) RemoveTrustedNode(trustedNodePublicAddress string) error {
	return ab.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(createKey(keyNotaryNodesPubAddress, []byte(trustedNodePublicAddress)))
	})
}

// CreateLeaf creates leaf vertex also known as a tip.
// Leaf validates previous leaf by:
// - checking leaf signature
// - checking leaf ancestors signatures to the depth of one parent
// - checking if leaf transferring spice issuer has enough founds.
// If leaf has valid signature and is referring to parent that is part of a graph then the leaf is valid.
// If leaf transfers spice then calculate issuer total founds and drain for given amount to calculate sufficient founds.
func (ab *AccountingBook) CreateLeaf(ctx context.Context, trx *transaction.Transaction) (Vertex, error) {
	ab.register()
	defer ab.unregister()

	leavesToExamine := 2
	var err error
	validatedLeafs := make([]Vertex, 0, 2)

	for _, item := range ab.dag.GetLeaves() {
		if leavesToExamine == 0 {
			break
		}

		var leaf Vertex
		switch vrx := item.(type) {
		case *Vertex:
			if vrx == nil {
				return Vertex{}, errors.Join(ErrUnexpected, errors.New("vertex is nil"))
			}
			leaf = *vrx
			err = ab.validateLeaf(ctx, &leaf)
			if err != nil {
				ab.dag.DeleteVertex(string(leaf.Hash[:]))
				ab.log.Error(
					fmt.Sprintf("Accounting book rejected leaf hash [ %v ], from [ %v ], %s",
						leaf.Hash, leaf.SignerPublicAddress, err),
				)
				continue
			}
		default:
			return Vertex{}, errors.Join(ErrUnexpected, errors.New("cannot match vertex type"))
		}

		leavesToExamine--

		validatedLeafs = append(validatedLeafs, leaf)
	}

	switch len(validatedLeafs) {
	case 2:
	case 1:
		rightHash := <-ab.lastVertexHash
		right, err := ab.dag.GetVertex(string(rightHash[:]))
		if err != nil {
			ab.log.Error(fmt.Sprintf("Accounting book create tip %s, %s", ErrUnexpected, err))
			return Vertex{}, ErrUnexpected
		}
		leafRight, ok := right.(*Vertex)
		if !ok {
			msgErr := errors.Join(ErrUnexpected, errors.New("right vertex type assertion failure"))
			ab.log.Error(fmt.Sprintf("Accounting book create tip %s.", msgErr))
			return Vertex{}, msgErr
		}
		validatedLeafs = append(validatedLeafs, *leafRight)

	case 0:
		rightHash := <-ab.lastVertexHash
		right, err := ab.dag.GetVertex(string(rightHash[:]))
		if err != nil {
			ab.log.Error(fmt.Sprintf("Accounting book create tip %s, %s", ErrUnexpected, err))
			return Vertex{}, ErrUnexpected
		}
		leafRight, ok := right.(*Vertex)
		if !ok {
			msgErr := errors.Join(ErrUnexpected, errors.New("right vertex type assertion failure"))
			ab.log.Error(fmt.Sprintf("Accounting book create tip %s.", msgErr))
			return Vertex{}, msgErr
		}
		validatedLeafs = append(validatedLeafs, *leafRight)

		leftHash := <-ab.lastVertexHash
		left, err := ab.dag.GetVertex(string(leftHash[:]))
		if err != nil {
			ab.log.Error(fmt.Sprintf("Accounting book create tip %s, %s", ErrUnexpected, err))
			return Vertex{}, ErrUnexpected
		}
		leafLeft, ok := left.(*Vertex)
		if !ok {
			msgErr := errors.Join(ErrUnexpected, errors.New("left vertex type assertion failure"))
			ab.log.Error(fmt.Sprintf("Accounting book create tip %s.", msgErr))
			return Vertex{}, msgErr
		}
		validatedLeafs = append(validatedLeafs, *leafLeft)

	default:
		msgErr := errors.Join(ErrUnexpected, fmt.Errorf("expected 2 vertexes got %v", len(validatedLeafs)))
		ab.log.Error(fmt.Sprintf("Accounting book create tip %s.", msgErr))
		return Vertex{}, msgErr
	}

	tip, err := NewVertex(*trx, validatedLeafs[0].Hash, validatedLeafs[1].Hash, ab.signer)
	if err != nil {
		ab.log.Error(fmt.Sprintf("Accounting book rejected new leaf [ %v ], %s.", tip.Hash, err))
		return Vertex{}, errors.Join(ErrNewLeafRejected, err)
	}
	if err := ab.dag.AddVertexByID(string(tip.Hash[:]), &tip); err != nil {
		ab.log.Error(fmt.Sprintf("Accounting book rejected new leaf [ %v ], %s.", tip.Hash, err))
		return Vertex{}, ErrNewLeafRejected
	}

	var isRoot bool
	for _, vrx := range validatedLeafs {
		ok, err := ab.dag.IsRoot(string(validatedLeafs[0].Hash[:]))
		if err != nil {
			ab.dag.DeleteVertex(string(tip.Hash[:]))
			ab.log.Error(
				fmt.Sprintf("Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ], %s,",
					vrx.Hash, vrx.SignerPublicAddress, vrx.LeftParentHash, vrx.RightParentHash, err),
			)
			return Vertex{}, ErrNewLeafRejected
		}
		if ok {
			if isRoot {
				continue
			}
			isRoot = true
		}
		if err := ab.dag.AddEdge(string(vrx.Hash[:]), string(tip.Hash[:])); err != nil {
			ab.dag.DeleteVertex(string(tip.Hash[:]))
			ab.log.Error(
				fmt.Sprintf("Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ], %s,",
					vrx.Hash, vrx.SignerPublicAddress, vrx.LeftParentHash, vrx.RightParentHash, err),
			)
			return Vertex{}, ErrNewLeafRejected
		}
	}
	for len(ab.lastVertexHash) > 0 {
		<-ab.lastVertexHash
	}
	for _, validVrx := range validatedLeafs {
		ab.lastVertexHash <- validVrx.Hash
	}

	return tip, nil
}

// AddLeaf adds leaf known also as tip to the graph for future validation.
// Leaf will be a subject of validation by another tip.
func (ab *AccountingBook) AddLeaf(ctx context.Context, leaf *Vertex) error {
	if leaf == nil {
		return ErrUnexpected
	}

	validatedLeafs := make([]Vertex, 0, 2)

	if err := leaf.verify(ab.verifier); err != nil {
		ab.log.Error(
			fmt.Sprintf("Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ], %s.",
				leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
		)
		return ErrLeafRejected
	}
	ab.register()
	defer ab.unregister()

	for _, hash := range [][32]byte{leaf.LeftParentHash, leaf.RightParentHash} {
		item, err := ab.dag.GetVertex(string(hash[:]))
		if err != nil {
			ab.log.Error(
				fmt.Sprintf("Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ], %s.",
					leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
			)
			return ErrLeafRejected
		}
		existringLeaf, ok := item.(Vertex)
		if !ok {
			return ErrUnexpected
		}
		isLeaf, err := ab.dag.IsLeaf(string(hash[:]))
		if err != nil {
			ab.log.Error(
				fmt.Sprintf("Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ], %s.",
					leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
			)
			return ErrLeafRejected
		}
		if isLeaf {
			if err := ab.validateLeaf(ctx, &existringLeaf); err != nil {
				return errors.Join(ErrLeafRejected, err)
			}
		}
		validatedLeafs = append(validatedLeafs, existringLeaf)
	}
	if err := ab.dag.AddVertexByID(string(leaf.Hash[:]), &leaf); err != nil {
		ab.log.Error(fmt.Sprintf("Accounting book rejected new leaf [ %v ], %s.", leaf.Hash, err))
		return ErrLeafRejected
	}

	for _, validVrx := range validatedLeafs {
		if err := ab.dag.AddEdge(string(validVrx.Hash[:]), string(leaf.Hash[:])); err != nil {
			ab.dag.DeleteVertex(string(leaf.Hash[:]))
			ab.log.Error(
				fmt.Sprintf("Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ], %s,",
					leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
			)
			return ErrLeafRejected
		}
	}
	for len(ab.lastVertexHash) > 0 {
		<-ab.lastVertexHash
	}
	for _, validVrx := range validatedLeafs {
		ab.lastVertexHash <- validVrx.Hash
	}

	return nil
}

// CalculateBalance traverses the graph starting from the recent accepted Vertex,
// and calculates the balance for the given address.
func (ab *AccountingBook) CalculateBalance(ctx context.Context, walletPubAddr string) (Balance, error) {
	lastVertexHash := <-ab.lastVertexHash
	item, err := ab.dag.GetVertex(string(lastVertexHash[:]))
	if err != nil {
		return Balance{}, errors.Join(ErrUnexpected, err)
	}

	spiceOut := spice.New(0, 0)
	spiceIn := spice.New(0, 0)
	switch vrx := item.(type) {
	case *Vertex:
		if vrx == nil {
			return Balance{}, ErrUnexpected
		}
		if err := pourFounds(walletPubAddr, *vrx, &spiceIn, &spiceOut); err != nil {
			return Balance{}, err
		}
	default:
		return Balance{}, ErrUnexpected

	}
	visited := make(map[string]struct{})
	vertices, signal, _ := ab.dag.AncestorsWalker(string(lastVertexHash[:]))
	for ancestorID := range vertices {
		select {
		case <-ctx.Done():
			signal <- true
			return Balance{}, ErrLeafBallanceCalculationProcessStopped
		default:
		}
		if _, ok := visited[ancestorID]; ok {
			continue
		}
		visited[ancestorID] = struct{}{}

		item, err := ab.dag.GetVertex(ancestorID)
		if err != nil {
			signal <- true
			return Balance{}, errors.Join(ErrUnexpected, err)
		}
		switch vrx := item.(type) {
		case *Vertex:
			if vrx == nil {
				return Balance{}, ErrUnexpected
			}
			if err := pourFounds(walletPubAddr, *vrx, &spiceIn, &spiceOut); err != nil {
				return Balance{}, err
			}

		default:
			signal <- true
			return Balance{}, ErrUnexpected
		}
	}

	s := spice.New(0, 0)
	if err := s.Supply(spiceIn); err != nil {
		return Balance{}, errors.Join(ErrBalanceCaclulationUnexpectedFailure, err)
	}

	if err := s.Drain(spiceOut, &spice.Melange{}); err != nil {
		return Balance{}, errors.Join(ErrBalanceCaclulationUnexpectedFailure, err)
	}

	return NewBalance(walletPubAddr, s), nil
}
