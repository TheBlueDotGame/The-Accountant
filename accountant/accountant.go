package accountant

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/dgraph-io/badger/v4"
	"github.com/heimdalr/dag"

	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/spice"
	"github.com/bartossh/Computantis/storage"
	"github.com/bartossh/Computantis/transaction"
)

const (
	initialThroughput     = 50
	throughputMultiplayer = 2
	lastVertexHashes      = 100
)

var (
	ErrGenesisRejected                       = errors.New("genesis vertex has been rejected")
	ErrBalanceCaclulationUnexpectedFailure   = errors.New("balance calculation unexpected failure")
	ErrBalanceUnavailable                    = errors.New("balance unavailable")
	ErrLeafBallanceCalculationProcessStopped = errors.New("wallet balance calculation process stopped")
	ErrLeafValidationProcessStopped          = errors.New("leaf validation process stopped")
	ErrNewLeafRejected                       = errors.New("new leaf rejected")
	ErrLeafRejected                          = errors.New("leaf rejected")
	ErrDagIsLoaded                           = errors.New("dag is already loaded")
	ErrDagIsNotLoaded                        = errors.New("dag is not loaded")
	ErrLeafAlreadyExists                     = errors.New("leaf already exists")
	ErrIssuerAddressBalanceNotFound          = errors.New("issuer address balance not found")
	ErrReceiverAddressBalanceNotFound        = errors.New("receiver address balance not found")
	ErrDoubleSpendingOrInsufficinetFounds    = errors.New("double spending or insufficient founds")
	ErrVertexHashNotFound                    = errors.New("vertex hash not found")
	ErrVertexAlreadyExists                   = errors.New("vertex already exists")
	ErrTrxInVertexAlreadyExists              = errors.New("transaction in vertex already exists")
	ErrTrxToVertexNotFound                   = errors.New("trx mapping to vertex do not found, transaction doesn't exist")
	ErrUnexpected                            = errors.New("unexpected failure")
	ErrTransferringFoundsFailure             = errors.New("transferring founds failure")
)

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
	trustedNodesDB *badger.DB
	tokensDB       *badger.DB
	trxsToVertxDB  *badger.DB
	verticesDB     *badger.DB
	lastVertexHash chan [32]byte
	registry       chan struct{}
	gennessisHash  [32]byte
	weight         atomic.Uint64
	throughput     atomic.Uint64
	dagLoaded      bool
}

// New creates new AccountingBook.
// New AccountingBook will start internally the garbage collection loop, to stop it from running cancel the context.
func NewAccountingBook(ctx context.Context, cfg Config, verifier signatureVerifier, signer signer, l logger.Logger) (*AccountingBook, error) {
	trustedNodesDB, err := storage.CreateBadgerDB(ctx, cfg.TrustedNodesDBPath, l)
	if err != nil {
		return nil, err
	}
	tokensDB, err := storage.CreateBadgerDB(ctx, cfg.TokensDBPath, l)
	if err != nil {
		return nil, err
	}
	trxsToVertxDB, err := storage.CreateBadgerDB(ctx, cfg.TraxsToVerticesMapDBPath, l)
	if err != nil {
		return nil, err
	}
	verticesDB, err := storage.CreateBadgerDB(ctx, cfg.VerticesDBPath, l)
	if err != nil {
		return nil, err
	}
	ab := &AccountingBook{
		verifier:       verifier,
		signer:         signer,
		dag:            dag.NewDAG(),
		trustedNodesDB: trustedNodesDB,
		tokensDB:       tokensDB,
		trxsToVertxDB:  trxsToVertxDB,
		verticesDB:     verticesDB,
		lastVertexHash: make(chan [32]byte, lastVertexHashes),
		registry:       make(chan struct{}, 1),
		log:            l,
		weight:         atomic.Uint64{},
		throughput:     atomic.Uint64{},
		dagLoaded:      !cfg.LoadDAG,
	}

	ab.unregister() // on new AccountingBook creation send to the register channel to unblock the register queue.

	return ab, nil
}

func (ab *AccountingBook) validateLeaf(ctx context.Context, leaf *Vertex) error {
	if leaf == nil {
		return errors.Join(ErrUnexpected, errors.New("leaf to validate is nil"))
	}
	if !ab.isValidWeight(leaf.Weight) {
		return errors.Join(
			ErrLeafRejected,
			fmt.Errorf("leaf doesn't meet condition of minimal weight, throughput: %v current: %v, received: %v", ab.throughput.Load(), ab.weight.Load(), leaf.Weight),
		)
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
	err := ab.trustedNodesDB.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(trustedNodePublicAddress))
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

func (ab *AccountingBook) checkTrxInVertexExists(trxHash []byte) (bool, error) {
	err := ab.trxsToVertxDB.View(func(txn *badger.Txn) error {
		_, err := txn.Get(trxHash)
		if err != nil {
			return err
		}
		return nil
	})
	if err == nil {
		return true, nil
	}
	switch err {
	case badger.ErrKeyNotFound:
		return false, nil
	default:
		ab.log.Error(fmt.Sprintf("transaction to vertex mapping for existing trx lookup failed, %s", err))
		return false, ErrUnexpected
	}
}

func (ab *AccountingBook) saveTrxInVertex(trxHash, vrxHash []byte) error {
	return ab.trxsToVertxDB.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(trxHash); err == nil {
			return ErrTrxInVertexAlreadyExists
		}
		return txn.SetEntry(badger.NewEntry(trxHash, vrxHash))
	})
}

func (ab *AccountingBook) removeTrxInVertex(trxHash []byte) error {
	return ab.trxsToVertxDB.Update(func(txn *badger.Txn) error {
		return txn.Delete(trxHash)
	})
}

func (ab *AccountingBook) readTrxVertex(trxHash []byte) (Vertex, error) {
	var vrxHash []byte
	err := ab.trxsToVertxDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get(trxHash)
		if err != nil {
			return err
		}
		item.Value(func(v []byte) error {
			vrxHash = v
			return nil
		})
		return nil
	})
	if err != nil {
		switch err {
		case badger.ErrKeyNotFound:
			return Vertex{}, ErrTrxToVertexNotFound
		default:
			ab.log.Error(fmt.Sprintf("transaction to vertex mapping failed when looking for transaction hash, %s", err))
			return Vertex{}, ErrUnexpected
		}
	}
	return ab.readVertex(vrxHash)
}

func (ab *AccountingBook) readVertex(vrxHash []byte) (Vertex, error) {
	vrx, err := ab.readVertexFromDAG(vrxHash)
	if err == nil {
		return vrx, nil
	}
	if !errors.Is(err, ErrVertexHashNotFound) {
		return Vertex{}, err
	}
	return ab.readVertexFromStorage(vrxHash)
}

func (ab *AccountingBook) checkVertexExists(vrxHash []byte) (bool, error) {
	_, err := ab.dag.GetVertex(string(vrxHash))
	if err == nil {
		return true, nil
	}
	err = ab.verticesDB.View(func(txn *badger.Txn) error {
		_, err := txn.Get(vrxHash)
		return err
	})
	if err != nil {
		switch err {
		case badger.ErrKeyNotFound:
			return false, nil
		default:
			ab.log.Error(fmt.Sprintf("transaction to vertex mapping failed when looking for transaction hash, %s", err))
			return false, ErrUnexpected
		}
	}
	return true, nil
}

func (ab *AccountingBook) readVertexFromDAG(vrxHash []byte) (Vertex, error) {
	item, err := ab.dag.GetVertex(string(vrxHash))
	if err == nil {
		switch v := item.(type) {
		case *Vertex:
			return *v, nil
		default:
			return Vertex{}, ErrUnexpected
		}
	}
	return Vertex{}, ErrVertexHashNotFound
}

func (ab *AccountingBook) readVertexFromStorage(vrxHash []byte) (Vertex, error) {
	var vrx Vertex
	err := ab.verticesDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get(vrxHash)
		if err != nil {
			return err
		}
		item.Value(func(v []byte) error {
			vrx, err = decodeVertex(v)
			return err
		})
		return nil
	})
	if err != nil {
		switch err {
		case badger.ErrKeyNotFound:
			return vrx, ErrVertexHashNotFound
		default:
			ab.log.Error(fmt.Sprintf("transaction to vertex mapping failed when looking for vertex hash, %s", err))
			return vrx, ErrUnexpected
		}
	}

	return vrx, nil
}

func (ab *AccountingBook) saveVertexToStorage(vrx *Vertex) error {
	buf, err := vrx.encode()
	if err != nil {
		return err
	}
	return ab.verticesDB.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(vrx.Hash[:]); err == nil {
			return ErrVertexAlreadyExists
		}
		return txn.SetEntry(badger.NewEntry(vrx.Hash[:], buf))
	})
}

func (ab *AccountingBook) updateWaightAndThroughput(weight uint64) {
	if ab.weight.Load() < weight {
		ab.weight.Store(weight)
	}
	leafsCount := uint64(len(ab.dag.GetLeaves()))
	ab.throughput.Store(((ab.throughput.Load() + leafsCount) / 2 * throughputMultiplayer))
}

func (ab *AccountingBook) isValidWeight(weight uint64) bool {
	current := ab.weight.Load()
	throughput := ab.throughput.Load()
	if throughput > current {
		return true
	}
	return weight >= current-throughput
}

// CreateGenesis creates genesis vertex that will transfer spice to current node as a receiver.
func (ab *AccountingBook) CreateGenesis(subject string, spc spice.Melange, data []byte, receiver signer) (Vertex, error) {
	ab.register()
	defer ab.unregister()
	trx, err := transaction.New(subject, spc, data, receiver.Address(), ab.signer)
	if err != nil {
		return Vertex{}, errors.Join(ErrGenesisRejected, err)
	}

	vrx, err := NewVertex(trx, [32]byte{}, [32]byte{}, 0, ab.signer)
	if err != nil {
		return Vertex{}, errors.Join(ErrGenesisRejected, err)
	}

	if err := ab.saveTrxInVertex(trx.Hash[:], vrx.Hash[:]); err != nil {
		return Vertex{}, errors.Join(ErrGenesisRejected, err)
	}

	if err := ab.dag.AddVertexByID(string(vrx.Hash[:]), &vrx); err != nil {
		return Vertex{}, err
	}

	ab.throughput.Store(initialThroughput)
	ab.updateWaightAndThroughput(initialThroughput)

	ab.lastVertexHash <- vrx.Hash
	ab.lastVertexHash <- vrx.Hash

	return vrx, nil
}

// LoadDag loads stream of Vertices in to the DAG.
func (ab *AccountingBook) LoadDag(ctx context.Context, cancelF context.CancelCauseFunc, cVrx <-chan *Vertex) {
	if ab.DagLoaded() {
		cancelF(ErrDagIsLoaded)
		return
	}
	ab.register()
	defer ab.unregister()
	defer ab.throughput.Store(initialThroughput)
	defer ab.updateWaightAndThroughput(initialThroughput)

	var counter int
	var lastLeafHash [32]byte

VertxLoop:
	for {
		select {
		case <-ctx.Done():
			break VertxLoop
		case vrx := <-cVrx:
			if vrx == nil {
				break VertxLoop
			}
			if err := ab.saveTrxInVertex(vrx.Transaction.Hash[:], vrx.Hash[:]); err != nil {
				cancelF(ErrLeafRejected)
				return
			}

			if err := ab.dag.AddVertexByID(string(vrx.Hash[:]), vrx); err != nil {
				cancelF(err)
				return
			}
			lastLeafHash = vrx.Hash
			counter++
		}
	}

	if counter == 0 {
		cancelF(ErrUnexpected)
		return
	}

	if counter == 1 {
		ab.lastVertexHash <- lastLeafHash
		ab.lastVertexHash <- lastLeafHash
		ab.dagLoaded = true
		return
	}

	for _, item := range ab.dag.GetVertices() {
		switch vrx := item.(type) {
		case *Vertex:
			for _, conn := range [][32]byte{vrx.LeftParentHash, vrx.RightParentHash} {
				if err := ab.dag.AddEdge(string(conn[:]), string(vrx.Hash[:])); err != nil {
					cancelF(err)
					return
				}
			}
		default:
			cancelF(ErrUnexpected)
			return
		}
	}

	for _, item := range ab.dag.GetLeaves() {
		switch vrx := item.(type) {
		case *Vertex:
			ab.lastVertexHash <- vrx.Hash
			lastLeafHash = vrx.Hash
		default:
			cancelF(ErrUnexpected)
			return
		}
	}

	if len(ab.lastVertexHash) == 0 {
		cancelF(ErrUnexpected)
		return
	}

	if len(ab.lastVertexHash) == 1 {
		ab.lastVertexHash <- lastLeafHash
	}
	ab.dagLoaded = true
}

func (ab *AccountingBook) DagLoaded() bool {
	return ab.dagLoaded
}

func (ab *AccountingBook) StreamDAG(ctx context.Context) (<-chan *Vertex, <-chan error) {
	cVrx := make(chan *Vertex, 100)
	cDone := make(chan error, 1)
	go func(cVrx chan<- *Vertex, cDone chan<- error) {
	VerticesLoop:
		for _, item := range ab.dag.GetVertices() {
			select {
			case <-ctx.Done():
				break VerticesLoop
			default:
			}
			switch vrx := item.(type) {
			case *Vertex:
				cVrx <- vrx
			default:
				cDone <- ErrUnexpected
				break VerticesLoop
			}
		}
		close(cDone)
		close(cVrx)
	}(cVrx, cDone)

	return cVrx, cDone
}

// AddTrustedNode adds trusted node public address to the trusted nodes public address repository.
func (ab *AccountingBook) AddTrustedNode(trustedNodePublicAddress string) error {
	return ab.trustedNodesDB.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(trustedNodePublicAddress), []byte{})
	})
}

// RemoveTrustedNode removes trusted node public address from trusted nodes public address repository.
func (ab *AccountingBook) RemoveTrustedNode(trustedNodePublicAddress string) error {
	return ab.trustedNodesDB.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(trustedNodePublicAddress))
	})
}

// CreateLeaf creates leaf vertex also known as a tip.
// All the graph validations before adding the leaf happens in that function,
// Created leaf will be a subject of validation by another tip.
func (ab *AccountingBook) CreateLeaf(ctx context.Context, trx *transaction.Transaction) (Vertex, error) {
	if !ab.DagLoaded() {
		return Vertex{}, ErrDagIsNotLoaded
	}
	ok, err := ab.checkTrxInVertexExists(trx.Hash[:])
	if err != nil {
		ab.log.Error(fmt.Sprintf("Accounting book creating transaction failed when checking trx to vertex mapping, %s", err))
		return Vertex{}, ErrUnexpected
	}
	if ok {
		return Vertex{}, ErrTrxInVertexAlreadyExists
	}

	ab.register()
	defer ab.unregister()

	leavesToExamine := 2
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
				ab.removeTrxInVertex(leaf.Transaction.Hash[:])
				ab.log.Error(
					fmt.Sprintf("Accounting book rejected leaf hash [ %v ], from [ %v ], %s",
						leaf.Hash, leaf.SignerPublicAddress, err),
				)
				ab.updateWaightAndThroughput(leaf.Weight)
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

	tip, err := NewVertex(
		*trx, validatedLeafs[0].Hash, validatedLeafs[1].Hash, calcNewWeight(validatedLeafs[0].Weight, validatedLeafs[1].Weight), ab.signer,
	)
	if err != nil {
		ab.log.Error(fmt.Sprintf("Accounting book rejected new leaf [ %v ], %s.", tip.Hash, err))
		return Vertex{}, errors.Join(ErrNewLeafRejected, err)
	}
	if err := ab.saveTrxInVertex(trx.Hash[:], tip.Hash[:]); err != nil {
		ab.log.Error(fmt.Sprintf("Accounting book vertex create failed saving transaction [ %v ] in tip [ %v ], %s.", trx.Hash[:], tip.Hash, err))
		return Vertex{}, ErrUnexpected
	}
	if err := ab.dag.AddVertexByID(string(tip.Hash[:]), &tip); err != nil {
		ab.removeTrxInVertex(trx.Hash[:])
		ab.log.Error(fmt.Sprintf("Accounting book rejected new leaf [ %v ], %s.", tip.Hash, err))
		return Vertex{}, ErrNewLeafRejected
	}

	var isRoot bool
	for _, vrx := range validatedLeafs {
		ok, err := ab.dag.IsRoot(string(validatedLeafs[0].Hash[:]))
		if err != nil {
			ab.dag.DeleteVertex(string(tip.Hash[:]))
			ab.removeTrxInVertex(trx.Hash[:])
			ab.log.Error(
				fmt.Sprintf("Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ] when checking is root, %s,",
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
			ab.removeTrxInVertex(trx.Hash[:])
			ab.log.Error(
				fmt.Sprintf("Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ] when adding an edge, %s,",
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
// Added leaf will be a subject of validation by another tip.
func (ab *AccountingBook) AddLeaf(ctx context.Context, leaf *Vertex) error {
	if !ab.DagLoaded() {
		return ErrDagIsNotLoaded
	}
	if leaf == nil {
		return errors.Join(ErrUnexpected, errors.New("leaf is nil"))
	}

	ok, err := ab.checkVertexExists(leaf.Hash[:])
	if err != nil {
		ab.log.Error(fmt.Sprintf("Accounting book adding leaf failed when checking vertex exists, %s", err))
		return errors.Join(ErrUnexpected, err)
	}
	if ok {
		return ErrLeafAlreadyExists
	}
	ok, err = ab.checkTrxInVertexExists(leaf.Transaction.Hash[:])
	if err != nil {
		ab.log.Error(fmt.Sprintf("Accounting book adding leaf failed when checking if trx to vertex mapping exists, %s", err))
		return errors.Join(ErrUnexpected, err)
	}
	if ok {
		return ErrTrxInVertexAlreadyExists
	}

	if err := leaf.verify(ab.verifier); err != nil {
		ab.log.Error(
			fmt.Sprintf("Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ] when verifying, %s.",
				leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
		)
		return ErrLeafRejected
	}
	ab.register()
	defer ab.unregister()

	validatedLeafs := make([]Vertex, 0, 2)

	for _, hash := range [][32]byte{leaf.LeftParentHash, leaf.RightParentHash} {
		item, err := ab.dag.GetVertex(string(hash[:]))
		if err != nil {
			ab.log.Error(
				fmt.Sprintf("Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ] when reading vertex, %s.",
					leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
			)
			return ErrLeafRejected
		}
		existringLeaf, ok := item.(*Vertex)
		if !ok {
			return errors.Join(ErrUnexpected, errors.New("wrong leaf type"))
		}
		isLeaf, err := ab.dag.IsLeaf(string(hash[:]))
		if err != nil {
			ab.log.Error(
				fmt.Sprintf("Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ] when validate is leaf, %s.",
					leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
			)
			return ErrLeafRejected
		}
		if isLeaf {
			if err := ab.validateLeaf(ctx, existringLeaf); err != nil {
				ab.dag.DeleteVertex(string(existringLeaf.Hash[:]))
				ab.removeTrxInVertex(existringLeaf.Transaction.Hash[:])
				return errors.Join(ErrLeafRejected, err)
			}
			ab.updateWaightAndThroughput(existringLeaf.Weight)
		}
		validatedLeafs = append(validatedLeafs, *existringLeaf)
	}

	if err := ab.saveTrxInVertex(leaf.Transaction.Hash[:], leaf.Hash[:]); err != nil {
		ab.log.Error(
			fmt.Sprintf("Accounting book leaf add failed saving transaction [ %v ] in leaf [ %v ], %s.", leaf.Transaction.Hash[:], leaf.Hash, err),
		)
		return errors.Join(ErrUnexpected, err)
	}

	if err := ab.dag.AddVertexByID(string(leaf.Hash[:]), leaf); err != nil {
		ab.log.Error(fmt.Sprintf("Accounting book rejected new leaf [ %v ], %s.", leaf.Hash, err))
		ab.removeTrxInVertex(leaf.Transaction.Hash[:])
		return ErrLeafRejected
	}

	var isRoot bool
	for _, validVrx := range validatedLeafs {
		ok, err := ab.dag.IsRoot(string(validatedLeafs[0].Hash[:]))
		if err != nil {
			ab.dag.DeleteVertex(string(leaf.Hash[:]))
			ab.removeTrxInVertex(leaf.Transaction.Hash[:])
			ab.log.Error(
				fmt.Sprintf("Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ] when checking is root, %s,",
					leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
			)
			return ErrNewLeafRejected
		}
		if ok {
			if isRoot {
				continue
			}
			isRoot = true
		}
		if err := ab.dag.AddEdge(string(validVrx.Hash[:]), string(leaf.Hash[:])); err != nil {
			ab.dag.DeleteVertex(string(leaf.Hash[:]))
			ab.removeTrxInVertex(leaf.Transaction.Hash[:])
			ab.log.Error(
				fmt.Sprintf("Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ] when adding edge, %s.",
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
	ab.register()
	defer ab.unregister()
	lastVertexHash := <-ab.lastVertexHash
	switch len(ab.lastVertexHash) {
	case 1:
		ab.lastVertexHash <- lastVertexHash
	case 0:
		ab.lastVertexHash <- lastVertexHash
		ab.lastVertexHash <- lastVertexHash
	default:
	}
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
