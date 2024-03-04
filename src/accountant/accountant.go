package accountant

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/heimdalr/dag"

	"github.com/bartossh/Computantis/src/logger"
	"github.com/bartossh/Computantis/src/spice"
	"github.com/bartossh/Computantis/src/transaction"
)

const (
	initialThroughput  uint64 = 50
	truncateVrxTopMark uint64 = 100_000
	trucateDiff        uint64 = 1_000
)

const repiterTick = time.Second * 2

const backupName = "vertex_db_backup_"

func checkCanTruncate(current, desired uint64) bool {
	return current > desired && current > trucateDiff && current-trucateDiff > desired
}

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
	ErrIssuerAddressBalanceNotfund           = errors.New("issuer address balance not fund")
	ErrReceiverAddressBalanceNotfund         = errors.New("receiver address balance not fund")
	ErrDoubleSpendingOrInsufficinetfunds     = errors.New("double spending or insufficient funds")
	ErrCannotTransferfundsViaOwnedNode       = errors.New("issuer cannot transfer funds via owned node")
	ErrCannotTransferfundsFromGenesisWallet  = errors.New("issuer cannot be the genesis node")
	ErrVertexHashNotfund                     = errors.New("vertex hash not fund")
	ErrVertexAlreadyExists                   = errors.New("vertex already exists")
	ErrTrxInVertexAlreadyExists              = errors.New("transaction in vertex already exists")
	ErrTrxToVertexNotfund                    = errors.New("trx mapping to vertex do not fund, transaction doesn't exist")
	ErrUnexpected                            = errors.New("unexpected failure")
	ErrTransferringfundsFailure              = errors.New("transferring spice failure")
	ErrEntityNotfund                         = errors.New("entity not fund")
	ErrBreak                                 = errors.New("just break")
)

type signatureVerifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

// Signer signs the given message and has a public address.
type Signer interface {
	Sign(message []byte) (digest [32]byte, signature []byte)
	Address() string
}

// AccountingBook is an entity that represents the accounting process of all received transactions.
type AccountingBook struct {
	truncateSignal       chan uint64
	repiter              *buffer
	verifier             signatureVerifier
	signer               Signer
	log                  logger.Logger
	dag                  *dag.DAG
	trustedNodesDB       *badger.DB
	trxsToVertxDB        *badger.DB
	verticesDB           *badger.DB
	genesisPublicAddress string
	mux                  sync.RWMutex
	weight               atomic.Uint64
	throughput           atomic.Uint64
	lastBackup           uint64
	nextWeightTruncate   uint64
	dagLoaded            bool
}

// New creates new AccountingBook.
// New AccountingBook will start internally the garbage collection loop, to stop it from running cancel the context.
func NewAccountingBook(
	ctx context.Context, cfg Config, verifier signatureVerifier, signer Signer, l logger.Logger,
) (*AccountingBook, error) {
	repi, err := newReplierBuffer(ctx, repiterTick)
	if err != nil {
		return nil, err
	}

	trustedNodesDB, err := createBadgerDB(ctx, cfg.TrustedNodesDBPath, l, true)
	if err != nil {
		return nil, err
	}
	trxsToVertxDB, err := createBadgerDB(ctx, cfg.TraxsToVerticesMapDBPath, l, true)
	if err != nil {
		return nil, err
	}
	verticesDB, err := createBadgerDB(ctx, cfg.VerticesDBPath, l, true)
	if err != nil {
		return nil, err
	}

	if cfg.Truncate < trucateDiff*2 {
		cfg.Truncate = truncateVrxTopMark
	}

	ab := &AccountingBook{
		truncateSignal:     make(chan uint64, initialThroughput),
		repiter:            repi,
		verifier:           verifier,
		signer:             signer,
		dag:                dag.NewDAG(),
		trustedNodesDB:     trustedNodesDB,
		trxsToVertxDB:      trxsToVertxDB,
		verticesDB:         verticesDB,
		mux:                sync.RWMutex{},
		log:                l,
		weight:             atomic.Uint64{},
		throughput:         atomic.Uint64{},
		lastBackup:         uint64(0),
		nextWeightTruncate: cfg.Truncate,
	}

	go ab.runLeafSubscriber(ctx)
	go ab.runTruncate(ctx)

	return ab, nil
}

func (ab *AccountingBook) runLeafSubscriber(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case v := <-ab.repiter.subscribe():
			if v.vrx == nil {
				continue
			}
			ab.addLeafMemorized(ctx, v)
		}
	}
}

func (ab *AccountingBook) runTruncate(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case w := <-ab.truncateSignal:
			if !checkCanTruncate(w, ab.nextWeightTruncate) {
				continue
			}
			ab.log.Info(fmt.Sprintf("Starting truncate at weight [ %v ]", w))
			if err := ab.truncate(ctx); err != nil {
				ab.log.Fatal(err.Error())
				return
			}
			ab.nextWeightTruncate += w
			ab.log.Info(fmt.Sprintf("Finnished truncate at weight [ %v ]", w))
		}
	}
}

func (ab *AccountingBook) truncate(ctx context.Context) error {
	ab.mux.Lock()
	defer ab.mux.Unlock()

	tempHashes := ab.dag.GetLeaves()
	if len(tempHashes) == 0 {
		return nil
	}
	var tempTopVrxHash string

	for id := range tempHashes {
		tempTopVrxHash = id
		break
	}

	h := newHashAtDepth(trucateDiff)

	if err := ab.performOnAncestorWalker(ctx, tempTopVrxHash, h.next); err != nil && errors.Is(err, ErrBreak) {
		return err
	}

	topVrxHash := h.getHash()

	f, err := os.Create(fmt.Sprintf("%s%v.bak", backupName, ab.lastBackup))
	if err != nil {
		return err
	}
	defer f.Close()

	lastBackup, err := ab.verticesDB.Backup(f, ab.lastBackup)
	if err != nil {
		return err
	}
	ab.lastBackup = lastBackup + 1

	fm := newfundsMemMap()

	if err := ab.forEachfundFromStorage(fm.set); err != nil {
		return err
	}

	perform := func(v *Vertex) error {
		if err := fm.nextVertex(v); err != nil {
			return err
		}
		if err := ab.saveVertexToStorage(v); err != nil {
			return err
		}
		return nil
	}

	if err := ab.performOnAncestorWalker(ctx, string(topVrxHash[:]), perform); err != nil && !errors.Is(err, ErrBreak) {
		return err
	}

	if err := fm.saveToStorage(ab.saveFundsToStorage); err != nil {
		return err
	}

	b := newStrBufer(ab.dag.GetSize())
	add := func(v *Vertex) error {
		b.add(string(v.Hash[:]))
		return nil
	}

	if err := ab.performOnAncestorWalker(ctx, string(topVrxHash[:]), add); err != nil && !errors.Is(err, ErrBreak) {
		return err
	}

	for {
		k, ok := b.next()
		if !ok {
			break
		}
		ab.dag.DeleteVertex(k)
	}

	return nil
}

func (ab *AccountingBook) performOnAncestorWalker(ctx context.Context, topVrxHash string, perform func(vrx *Vertex) error) error {
	visited := make(map[string]struct{})
	vertices, signal, err := ab.dag.AncestorsWalker(topVrxHash)
	if err != nil {
		return err
	}
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
				signal <- true
				return ErrUnexpected
			}
			if err := perform(vrx); err != nil {
				signal <- true
				if err == ErrBreak {
					return nil
				}
				return errors.Join(ErrUnexpected, err)
			}

		default:
			signal <- true
			return ErrUnexpected
		}
	}

	return nil
}

func (ab *AccountingBook) validateLeaf(ctx context.Context, leaf *Vertex) error {
	if leaf == nil {
		return errors.Join(ErrUnexpected, errors.New("leaf to validate is nil"))
	}
	if !ab.isValidWeight(leaf.Weight) {
		return errors.Join(
			ErrLeafRejected,
			fmt.Errorf("leaf doesn't meet condition of minimal weight, throughput: %v current: %v, received: %v",
				ab.throughput.Load(), ab.weight.Load(), leaf.Weight,
			),
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

	s, err := ab.readAddressFundsFromStorage(leaf.Transaction.IssuerAddress)
	if err != nil && err != ErrBalanceUnavailable {
		return err
	}
	if err := spiceIn.Supply(s); err != nil {
		return err
	}

	if err := pourfunds(leaf.Transaction.IssuerAddress, *leaf, &spiceIn, &spiceOut); err != nil {
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
			if err := pourfunds(leaf.Transaction.IssuerAddress, *vrx, &spiceIn, &spiceOut); err != nil {
				return errors.Join(ErrTransferringfundsFailure, err)
			}

		default:
			signal <- true
			return ErrUnexpected
		}
	}

	err = checkHasSufficientfunds(&spiceIn, &spiceOut)
	if err != nil {
		ab.log.Info(
			fmt.Sprintf(
				"No sufficient funds [ in: %s ] [ out: %s ]\n",
				spiceIn, spiceOut,
			),
		)
		return errors.Join(ErrTransferringfundsFailure, err)
	}
	return nil
}

func (ab *AccountingBook) readVertex(vrxHash []byte) (Vertex, error) {
	vrx, err := ab.readVertexFromDAG(vrxHash)
	if err == nil {
		return vrx, nil
	}
	if !errors.Is(err, ErrVertexHashNotfund) {
		return Vertex{}, err
	}
	return ab.readVertexFromStorage(vrxHash)
}

func (ab *AccountingBook) checkVertexExists(vrxHash []byte) (bool, error) {
	_, err := ab.dag.GetVertex(string(vrxHash))
	if err == nil {
		return true, nil
	}
	return ab.checkVertexExistInStorage(vrxHash)
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
	return Vertex{}, ErrVertexHashNotfund
}

func (ab *AccountingBook) updateWaightAndThroughput(weight uint64) {
	if ab.weight.Load() < weight {
		ab.weight.Store(weight)
	}
	leafsCount := uint64(len(ab.dag.GetLeaves()))
	ab.throughput.Store(ab.throughput.Load() + leafsCount + 1)
}

func (ab *AccountingBook) isValidWeight(weight uint64) bool {
	current := ab.weight.Load()
	throughput := ab.throughput.Load()
	if throughput > current {
		return true
	}
	return weight >= current-throughput
}

func (ab *AccountingBook) getValidLeaves(ctx context.Context) (leftLeaf, rightLeaf *Vertex, err error) {
	var i int
	for _, item := range ab.dag.GetLeaves() {
		if i == 2 {
			break
		}

		switch vrx := item.(type) {
		case *Vertex:
			if vrx == nil {
				err = errors.Join(ErrUnexpected, errors.New("vertex is nil"))
				return
			}
			err = ab.validateLeaf(ctx, vrx)
			if err != nil {
				ab.dag.DeleteVertex(string(vrx.Hash[:]))
				ab.removeTrxInVertex(vrx.Transaction.Hash[:])
				ab.log.Error(
					fmt.Sprintf("Accounting book rejected leaf hash [ %v ], from [ %v ], %s",
						vrx.Hash, vrx.SignerPublicAddress, err),
				)
				ab.updateWaightAndThroughput(vrx.Weight)
				continue
			}
			switch i {
			case 0:
				leftLeaf = vrx
			case 1:
				rightLeaf = vrx
			}
			i++

		default:
			err = errors.Join(ErrUnexpected, errors.New("cannot match vertex type"))
			return
		}
	}
	return
}

func (ab *AccountingBook) addLeafMemorized(ctx context.Context, m memory) error {
	leaf := m.vrx
	if leaf == nil {
		return errors.Join(ErrUnexpected, errors.New("leaf is nil"))
	}
	if leaf.Transaction.IssuerAddress == ab.genesisPublicAddress {
		return ErrCannotTransferfundsFromGenesisWallet
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
			fmt.Sprintf(
				"Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ] when verifying, %s.",
				leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
		)
		return ErrLeafRejected
	}

	ab.mux.Lock()
	defer ab.mux.Unlock()

	validatedLeafs := make([]*Vertex, 0, 2)

	for _, hash := range [][32]byte{leaf.LeftParentHash, leaf.RightParentHash} {
		item, err := ab.dag.GetVertex(string(hash[:]))
		if err != nil {
			ab.log.Info(
				fmt.Sprintf(
					"Accounting book proceeded with memorizing leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ] when reading vertex for future validation, %s.",
					leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
			)
			if err := ab.repiter.insert(m); err != nil {
				ab.log.Error(
					fmt.Sprintf(
						"Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ] when reading vertex, %s.",
						leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
				)
				return ErrLeafRejected
			}
			return nil
		}
		existringLeaf, ok := item.(*Vertex)
		if !ok {
			return errors.Join(ErrUnexpected, errors.New("wrong leaf type"))
		}
		isLeaf, err := ab.dag.IsLeaf(string(hash[:]))
		if err != nil {
			ab.log.Error(
				fmt.Sprintf(
					"Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ] when validate is leaf, %s.",
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
		validatedLeafs = append(validatedLeafs, existringLeaf)
	}

	if err := ab.saveTrxInVertex(leaf.Transaction.Hash[:], leaf.Hash[:]); err != nil {
		ab.log.Error(
			fmt.Sprintf(
				"Accounting book leaf add failed saving transaction [ %v ] in leaf [ %v ], %s.",
				leaf.Transaction.Hash[:], leaf.Hash, err,
			),
		)
		return errors.Join(ErrUnexpected, err)
	}

	if err := ab.dag.AddVertexByID(string(leaf.Hash[:]), leaf); err != nil {
		ab.log.Error(fmt.Sprintf("Accounting book rejected new leaf [ %v ], %s.", leaf.Hash, err))
		ab.removeTrxInVertex(leaf.Transaction.Hash[:])
		return ErrLeafRejected
	}

	var addedHash [32]byte
	for _, validVrx := range validatedLeafs {
		if validVrx.Hash == addedHash {
			break
		}
		if err := ab.dag.AddEdge(string(validVrx.Hash[:]), string(leaf.Hash[:])); err != nil {
			ab.dag.DeleteVertex(string(leaf.Hash[:]))
			ab.removeTrxInVertex(leaf.Transaction.Hash[:])
			ab.log.Error(
				fmt.Sprintf(
					"Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ] when adding edge, %s.",
					leaf.Hash, leaf.SignerPublicAddress, leaf.LeftParentHash, leaf.RightParentHash, err),
			)
			return ErrLeafRejected
		}
		addedHash = validVrx.Hash
	}

	ab.truncateSignal <- leaf.Weight

	return nil
}

// CreateGenesis creates genesis vertex that will transfer spice to current node as a receiver.
func (ab *AccountingBook) CreateGenesis(subject string, spc spice.Melange, data []byte, reciverPublicAddress string) (Vertex, error) {
	if reciverPublicAddress == ab.signer.Address() {
		return Vertex{}, errors.Join(ErrGenesisRejected, errors.New("receiver and issuer cannot be the same wallet"))
	}

	trx, err := transaction.New(subject, spc, data, reciverPublicAddress, ab.signer)
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

	ab.mux.Lock()
	defer ab.mux.Unlock()

	if err := ab.dag.AddVertexByID(string(vrx.Hash[:]), &vrx); err != nil {
		return Vertex{}, err
	}

	ab.throughput.Store(initialThroughput)
	ab.updateWaightAndThroughput(initialThroughput)

	ab.dagLoaded = true
	ab.genesisPublicAddress = ab.signer.Address()

	return vrx, nil
}

// LoadDag loads stream of Vertices in to the DAG.
func (ab *AccountingBook) LoadDag(cancelF context.CancelCauseFunc, cVrx <-chan *Vertex) {
	if ab.DagLoaded() {
		cancelF(ErrDagIsLoaded)
		return
	}

	defer ab.throughput.Store(initialThroughput)
	defer ab.updateWaightAndThroughput(initialThroughput)

	ab.mux.Lock()
	defer ab.mux.Unlock()

VertxLoop:
	for vrx := range cVrx {
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
	}

	var maxWeight uint64
	var lastVrx *Vertex
	for _, item := range ab.dag.GetVertices() {
		switch vrx := item.(type) {
		case *Vertex:
			if vrx == nil {
				cancelF(ErrUnexpected)
				return
			}
			if vrx.Weight > maxWeight {
				maxWeight = vrx.Weight
			}
			var addedHash [32]byte
			lastVrx = vrx
		connLoop:
			for _, conn := range [][32]byte{vrx.LeftParentHash, vrx.RightParentHash} {
				if conn == addedHash {
					break connLoop
				}
				if err := ab.dag.AddEdge(string(conn[:]), string(vrx.Hash[:])); err != nil {
					cancelF(err)
					return
				}
				addedHash = conn
			}
		default:
			cancelF(ErrUnexpected)
			return
		}
	}

	ab.dagLoaded = true
	ab.genesisPublicAddress = lastVrx.Transaction.IssuerAddress
	ab.nextWeightTruncate = ab.nextWeightTruncate + maxWeight

	ab.log.Info(fmt.Sprintf("Loaded dag with success to weight [ %v ], total DAG size [ %v ]", maxWeight, ab.dag.GetSize()))
}

// DagLoaded returns true if dag is loaded or false otherwise.
func (ab *AccountingBook) DagLoaded() bool {
	return ab.dagLoaded
}

// StreamDAG provides tow channels to subscribe to a stream of vertices.
// First streams verticies and second one streams possible errors.
func (ab *AccountingBook) StreamDAG(ctx context.Context) <-chan *Vertex {
	ab.mux.RLock()
	defer ab.mux.RUnlock()

	cVrx := make(chan *Vertex, 100)
	go func(cVrx chan<- *Vertex) {
		visited := make(map[string]struct{})
	leavesLoop:
		for l := range ab.dag.GetLeaves() {
			select {
			case <-ctx.Done():
				break leavesLoop
			default:
			}
			item, err := ab.dag.GetVertex(l)
			if err != nil {
				break leavesLoop
			}
			switch vrx := item.(type) {
			case *Vertex:
				cVrx <- vrx
			default:
				break leavesLoop
			}
			vertices, signal, err := ab.dag.AncestorsWalker(l)
			if err != nil {
				signal <- true
				break leavesLoop
			}

		verticesLoop:
			for ancestorID := range vertices {
				select {
				case <-ctx.Done():
					signal <- true
					break leavesLoop
				default:
				}
				if _, ok := visited[ancestorID]; ok {
					continue verticesLoop
				}
				visited[ancestorID] = struct{}{}

				item, err := ab.dag.GetVertex(ancestorID)
				if err != nil {
					signal <- true
					break leavesLoop
				}
				switch vrx := item.(type) {
				case *Vertex:
					if vrx == nil {
						break leavesLoop
					}
					cVrx <- vrx
				default:
					signal <- true
					break leavesLoop
				}
			}

		}
		close(cVrx)
	}(cVrx)

	return cVrx
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
	if trx.IssuerAddress == ab.signer.Address() {
		return Vertex{}, ErrCannotTransferfundsViaOwnedNode
	}
	if trx.IssuerAddress == ab.genesisPublicAddress {
		return Vertex{}, ErrCannotTransferfundsFromGenesisWallet
	}

	ok, err := ab.checkTrxInVertexExists(trx.Hash[:])
	if err != nil {
		ab.log.Error(fmt.Sprintf(
			"Accounting book creating transaction failed when checking trx to vertex mapping, %s", err,
		),
		)
		return Vertex{}, ErrUnexpected
	}
	if ok {
		return Vertex{}, ErrTrxInVertexAlreadyExists
	}

	ab.mux.Lock()
	defer ab.mux.Unlock()

	leftLeaf, rightLeaf, err := ab.getValidLeaves(ctx)
	if err != nil {
		return Vertex{}, err
	}

	if leftLeaf == nil {
		leftLeaf, rightLeaf, err = ab.getValidLeaves(ctx)
		if err != nil {
			return Vertex{}, err
		}
		if leftLeaf != nil {
			msgErr := errors.Join(ErrUnexpected, errors.New("expected at least one leaf but got zero"))
			ab.log.Error(fmt.Sprintf("Accounting book create tip %s.", msgErr))
			return Vertex{}, msgErr
		}
	}

	if rightLeaf == nil {
		rightLeaf = leftLeaf
	}

	tip, err := NewVertex(
		*trx, leftLeaf.Hash, rightLeaf.Hash,
		calcNewWeight(leftLeaf.Weight, rightLeaf.Weight), ab.signer,
	)
	if err != nil {
		ab.log.Error(fmt.Sprintf("Accounting book rejected new leaf [ %v ], %s.", tip.Hash, err))
		return Vertex{}, errors.Join(ErrNewLeafRejected, err)
	}
	if err := ab.saveTrxInVertex(trx.Hash[:], tip.Hash[:]); err != nil {
		ab.log.Error(
			fmt.Sprintf(
				"Accounting book vertex create failed saving transaction [ %v ] in tip [ %v ], %s.",
				trx.Hash[:], tip.Hash, err,
			),
		)
		return Vertex{}, ErrUnexpected
	}
	if err := ab.dag.AddVertexByID(string(tip.Hash[:]), &tip); err != nil {
		ab.removeTrxInVertex(trx.Hash[:])
		ab.log.Error(fmt.Sprintf("Accounting book rejected new leaf [ %v ], %s.", tip.Hash, err))
		return Vertex{}, ErrNewLeafRejected
	}

	var addedHash [32]byte
	for _, vrx := range []*Vertex{leftLeaf, rightLeaf} {
		if vrx.Hash == addedHash {
			break
		}
		if err := ab.dag.AddEdge(string(vrx.Hash[:]), string(tip.Hash[:])); err != nil {
			ab.dag.DeleteVertex(string(tip.Hash[:]))
			ab.removeTrxInVertex(trx.Hash[:])
			ab.log.Error(
				fmt.Sprintf(
					"Accounting book rejected leaf [ %v ] from [ %v ] referring to [ %v ] and [ %v ] when adding an edge, %s,",
					vrx.Hash, vrx.SignerPublicAddress, vrx.LeftParentHash, vrx.RightParentHash, err),
			)
			return Vertex{}, ErrNewLeafRejected
		}
		addedHash = vrx.Hash
	}
	ab.truncateSignal <- tip.Weight
	return tip, nil
}

// AddLeaf adds leaf known also as tip to the graph for future validation.
// Added leaf will be a subject of validation by another tip.
func (ab *AccountingBook) AddLeaf(ctx context.Context, leaf *Vertex) error {
	if !ab.DagLoaded() {
		return ErrDagIsNotLoaded
	}
	return ab.addLeafMemorized(ctx, newMemory(leaf))
}

// CalculateBalance traverses the graph starting from the recent accepted Vertex,
// and calculates the balance for the given address.
func (ab *AccountingBook) CalculateBalance(ctx context.Context, walletPubAddr string) (Balance, error) {
	ab.mux.RLock()
	defer ab.mux.RUnlock()

	var leaf *Vertex
	var ok bool
	for _, item := range ab.dag.GetLeaves() {
		leaf, ok = item.(*Vertex)
		if !ok {
			return Balance{}, errors.Join(ErrUnexpected, errors.New("calculate balance, cannot cast item to leaf"))
		}
	}
	if leaf == nil {
		return Balance{}, errors.Join(ErrUnexpected, errors.New("calculate balance, cannot read leaf"))
	}

	item, err := ab.dag.GetVertex(string(leaf.Hash[:]))
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
		if err := pourfunds(walletPubAddr, *vrx, &spiceIn, &spiceOut); err != nil {
			return Balance{}, err
		}
	default:
		return Balance{}, ErrUnexpected

	}
	visited := make(map[string]struct{})
	vertices, signal, err := ab.dag.AncestorsWalker(string(leaf.Hash[:]))
	if err != nil {
		return Balance{}, errors.Join(ErrUnexpected, err)
	}
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
			if err := pourfunds(walletPubAddr, *vrx, &spiceIn, &spiceOut); err != nil {
				return Balance{}, err
			}
		default:
			signal <- true
			return Balance{}, ErrUnexpected
		}
	}

	s, err := ab.readAddressFundsFromStorage(walletPubAddr)
	if err != nil && err != ErrBalanceUnavailable {
		return Balance{}, err
	}
	if err := s.Supply(spiceIn); err != nil {
		return Balance{}, errors.Join(ErrBalanceCaclulationUnexpectedFailure, err)
	}

	if err := s.Drain(spiceOut, &spice.Melange{}); err != nil {
		return Balance{}, errors.Join(ErrBalanceCaclulationUnexpectedFailure, err)
	}

	return NewBalance(walletPubAddr, s), nil
}

// ReadTransactionByHash  reads transactions by hashes from DAG and DB.
func (ab *AccountingBook) ReadTransactionByHash(ctx context.Context, hash [32]byte) (transaction.Transaction, error) {
	vertexHash, err := ab.readVertexHashContainingTrxHashFromStorage(hash)
	if err != nil {
		return transaction.Transaction{}, err
	}

	ab.mux.RLock()
	defer ab.mux.RUnlock()

	item, err := ab.dag.GetVertex(string(vertexHash))
	switch err {
	case nil:
		switch vrx := item.(type) {
		case *Vertex:
			if vrx == nil {
				return transaction.Transaction{}, ErrUnexpected
			}
			return vrx.Transaction, nil // success
		default:
			return transaction.Transaction{}, ErrUnexpected
		}
	default:
		if !errors.Is(err, dag.IDUnknownError{}) {
			ab.log.Error(fmt.Sprintf("accountant error with reading vertex from DAG, %s", err))
		}
	}

	return ab.readTransactionFromStorage(vertexHash) // success
}

// ReadDAGTransactionsByAddress reads all the transactions from DAG only, that given address appears in as issuer or receiver.
// This will not read transactions after DAG has been truncated.
func (ab *AccountingBook) ReadDAGTransactionsByAddress(ctx context.Context, address string) ([]transaction.Transaction, error) {
	ab.mux.RLock()
	defer ab.mux.RUnlock()

	var leaf *Vertex
	var ok bool
	for _, item := range ab.dag.GetLeaves() {
		leaf, ok = item.(*Vertex)
		if !ok {
			return nil, errors.Join(ErrUnexpected, errors.New("reading transactions failed due to wrong leaf type"))
		}
	}
	if leaf == nil {
		return nil, errors.Join(ErrUnexpected, errors.New("reading transactions received nil leaf"))
	}

	var trasnsactions []transaction.Transaction

	item, err := ab.dag.GetVertex(string(leaf.Hash[:]))
	if err != nil {
		return nil, errors.Join(ErrUnexpected, err)
	}
	switch vrx := item.(type) {
	case *Vertex:
		if vrx == nil {
			return nil, ErrUnexpected
		}
		if vrx.Transaction.ReceiverAddress == address || vrx.Transaction.IssuerAddress == address {
			trasnsactions = append(trasnsactions, vrx.Transaction)
		}

	default:
		return nil, ErrUnexpected

	}
	visited := make(map[string]struct{})
	vertices, signal, err := ab.dag.AncestorsWalker(string(leaf.Hash[:]))
	if err != nil {
		return nil, errors.Join(ErrUnexpected, err)
	}
	for ancestorID := range vertices {
		select {
		case <-ctx.Done():
			signal <- true
			return nil, ErrLeafBallanceCalculationProcessStopped
		default:
		}
		if _, ok := visited[ancestorID]; ok {
			continue
		}
		visited[ancestorID] = struct{}{}

		item, err := ab.dag.GetVertex(ancestorID)
		if err != nil {
			signal <- true
			return nil, errors.Join(ErrUnexpected, err)
		}
		switch vrx := item.(type) {
		case *Vertex:
			if vrx == nil {
				return nil, ErrUnexpected
			}
			if vrx.Transaction.ReceiverAddress == address || vrx.Transaction.IssuerAddress == address {
				trasnsactions = append(trasnsactions, vrx.Transaction)
			}

		default:
			signal <- true
			return nil, ErrUnexpected
		}
	}

	return trasnsactions, nil
}

// Address returns signer public address that is a core cryptographic padlock for the DAG Vertices.
func (ab *AccountingBook) Address() string {
	return ab.signer.Address()
}
