package gossip

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/bartossh/Computantis/accountant"
	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/protobufcompiled"
	"github.com/bartossh/Computantis/spice"
	"github.com/bartossh/Computantis/storage"
	"github.com/bartossh/Computantis/transaction"
	"github.com/bartossh/Computantis/versioning"
	"github.com/dgraph-io/badger/v4"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	ErrDiscoveryAttmeptSignatureFailed                = errors.New("discovery attempt failed, signature or digest is invalid")
	ErrDiscoveryAttmeptRqustingNodeServerNotReachable = errors.New("discovery attempt failed, requesting node server not reachable")
	ErrUnexpectedGossipFailure                        = errors.New("unexpected gossip failure")
	ErrVertexInCache                                  = errors.New("vertex in cache")
	ErrFailedToProcessGossip                          = errors.New("failed to process gossip")
	ErrNilVertex                                      = errors.New("vertex is nil")
	ErrNilVertexGossip                                = errors.New("vertex gossip is nil")
)

const (
	vertexCacheLongevity = time.Second * 10
)

const (
	vertexGossipChCapacity = 5000
)

type nodeData struct {
	conn   *grpc.ClientConn
	client protobufcompiled.GossipAPIClient
	url    string
}

// Config is a configuration for the gossip node.
type Config struct {
	URL            string        `yaml:"url"`
	GenesisURL     string        `yaml:"genesis_url"`
	LoadDagURL     string        `yaml:"load_dag_url"`
	VerticesDBPath string        `yaml:"vertices_db_path"`
	GenesisSpice   spice.Melange `yaml:"genesis_spice"`
	Port           int           `yaml:"port"`
}

func (c Config) verify() error {
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("allowed port range is 0 to 65535, got %v", c.Port)
	}
	if _, err := url.Parse(c.GenesisURL); err != nil {
		return fmt.Errorf("cannot parse given genesis URL: [ %s ], %w", c.GenesisURL, err)
	}
	if _, err := url.Parse(c.URL); err != nil {
		return fmt.Errorf("cannot parse given node URL: [ %s ], %w", c.URL, err)
	}
	if _, err := url.Parse(c.LoadDagURL); err != nil {
		return fmt.Errorf("cannot parse given node URL: [ %s ], %w", c.URL, err)
	}
	return nil
}

type signatureVerifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

type accounter interface {
	CreateGenesis(subject string, spc spice.Melange, data []byte, receiver accountant.Signer) (accountant.Vertex, error)
	AddLeaf(ctx context.Context, leaf *accountant.Vertex) error
	StreamDAG(ctx context.Context) (<-chan *accountant.Vertex, <-chan error)
	LoadDag(ctx context.Context, cancelF context.CancelCauseFunc, cVrx <-chan *accountant.Vertex)
	DagLoaded() bool
}

type gossiper struct {
	protobufcompiled.UnimplementedGossipAPIServer
	accounter                accounter
	verifier                 signatureVerifier
	signer                   accountant.Signer
	log                      logger.Logger
	vertexCache              *badger.DB
	vrxCh                    <-chan *accountant.Vertex
	vertexGossipCh           chan *protobufcompiled.VertexGossip
	vertexGossipTimeSortedCh chan *protobufcompiled.VertexGossip
	nodes                    map[string]nodeData
	url                      string
	mux                      sync.RWMutex
	timeout                  time.Duration
}

// RunGRPC runs the service application that exposes the GRPC API for gossip protocol.
// To stop server cancel the context.
func RunGRPC(ctx context.Context, cfg Config, l logger.Logger, t time.Duration, s accountant.Signer,
	v signatureVerifier, a accounter, vrxCh <-chan *accountant.Vertex,
) error {
	if err := cfg.verify(); err != nil {
		return err
	}

	ctxx, cancel := context.WithCancel(ctx)
	defer cancel()

	db, err := storage.CreateBadgerDB(ctx, cfg.VerticesDBPath, l, true)
	if err != nil {
		return err
	}

	g := gossiper{
		accounter:                a,
		verifier:                 v,
		signer:                   s,
		log:                      l,
		vertexCache:              db,
		vrxCh:                    vrxCh,
		vertexGossipCh:           make(chan *protobufcompiled.VertexGossip, vertexGossipChCapacity),
		vertexGossipTimeSortedCh: make(chan *protobufcompiled.VertexGossip, 1),
		nodes:                    make(map[string]nodeData),
		url:                      cfg.URL,
		mux:                      sync.RWMutex{},
		timeout:                  t,
	}

	switch cfg.LoadDagURL {
	case "":
		g.accounter.CreateGenesis("Genesis Vertex", spice.New(cfg.GenesisSpice.Currency, cfg.GenesisSpice.SupplementaryCurrency), []byte{}, s)
	default:
		if err := g.updateDag(ctx, cfg.LoadDagURL); err != nil {
			cancel()
			g.log.Error(fmt.Sprintf("Failed loading DAG: %s", err))
			return err
		}
		g.log.Info(fmt.Sprintf("Node %s loaded DAG from URL: %s.", g.signer.Address(), cfg.URL))
	}

	defer g.closeAllNodesConnections()
	defer close(g.vertexGossipCh)

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%v", cfg.Port))
	if err != nil {
		cancel()
		return err
	}

	grpcServer := grpc.NewServer()
	protobufcompiled.RegisterGossipAPIServer(grpcServer, &g)

	go g.runProcessVertexGossip(ctx)
	go g.runTimeSortVertexGossip(ctx, cancel)

	go func() {
		err = grpcServer.Serve(lis)
		if err != nil {
			g.log.Fatal(fmt.Sprintf("Node [ %s ] cannot start server on port [ %v ], %s", s.Address(), cfg.Port, err))
			cancel()
		}
	}()

	time.Sleep(time.Millisecond * 50) // just wait so the server can start

	defer grpcServer.GracefulStop()

	if err == nil {
		if err := g.updateNodesConnectionsFromGensisNode(ctx, cfg.GenesisURL); err != nil {
			g.log.Fatal(
				fmt.Sprintf("Updating nodes on genesis URL [ %s ] for node [ %s ] failed, %s",
					cfg.GenesisURL, g.signer.Address(), err.Error(),
				),
			)
			cancel()
		}
	}

	g.log.Info(fmt.Sprintf("Server started on port [ %v ] for node [ %s ].", cfg.Port, s.Address()))

	<-ctxx.Done()

	return nil
}

func (g *gossiper) Alive(_ context.Context, _ *emptypb.Empty) (*protobufcompiled.AliveData, error) {
	return &protobufcompiled.AliveData{
		ApiVersion:    versioning.ApiVersion,
		ApiHeader:     versioning.Header,
		PublicAddress: g.signer.Address(),
	}, nil
}

func (g *gossiper) Announce(_ context.Context, cd *protobufcompiled.ConnectionData) (*emptypb.Empty, error) {
	err := g.valiudateSignature(cd.PublicAddress, cd.PublicAddress, cd.Url, cd.CreatedAt, cd.Signature, [32]byte(cd.Digest))
	if err != nil {
		g.log.Info(fmt.Sprintf("Discovery attempt failed, public address [ %s ] with URL [ %s ], %s", cd.PublicAddress, cd.Url, err))
		return nil, ErrDiscoveryAttmeptSignatureFailed
	}
	g.mux.Lock()
	defer g.mux.Unlock()
	if n, ok := g.nodes[cd.PublicAddress]; ok {
		n.conn.Close()
		delete(g.nodes, cd.PublicAddress)
	}
	nd, err := connectToNode(cd.Url)
	if err != nil {
		return nil, err
	}

	g.nodes[cd.PublicAddress] = nd
	g.log.Info(fmt.Sprintf("Node [ %s ] connected to [ %s ] with URL [ %s ].", g.signer.Address(), cd.PublicAddress, cd.Url))

	return &emptypb.Empty{}, nil
}

func (g *gossiper) Discover(_ context.Context, cd *protobufcompiled.ConnectionData) (*protobufcompiled.ConnectedNodes, error) {
	err := g.valiudateSignature(cd.PublicAddress, cd.PublicAddress, cd.Url, cd.CreatedAt, cd.Signature, [32]byte(cd.Digest))
	if err != nil {
		g.log.Info(fmt.Sprintf("Discovery attempt failed, public address [ %s ] with URL [ %s ], %s", cd.PublicAddress, cd.Url, err))
		return nil, ErrDiscoveryAttmeptSignatureFailed
	}

	nd, err := connectToNode(cd.Url)
	if err != nil {
		return nil, err
	}

	g.mux.Lock()
	defer g.mux.Unlock()
	if n, ok := g.nodes[cd.PublicAddress]; ok {
		n.conn.Close()
		delete(g.nodes, cd.PublicAddress)
	}

	g.nodes[cd.PublicAddress] = nd
	g.log.Info(fmt.Sprintf("Node [ %s ] connected to [ %s ] with URL [ %s ].", g.signer.Address(), cd.PublicAddress, cd.Url))

	connected := &protobufcompiled.ConnectedNodes{
		SignerPublicAddress: g.signer.Address(),
		Connections:         make([]*protobufcompiled.ConnectionData, 0, len(g.nodes)+1),
	}
	now := uint64(time.Now().UnixNano())
	data := initConnectionData(g.signer.Address(), g.url, now)
	digest, signature := g.signer.Sign(data)
	connected.Connections = append(connected.Connections, &protobufcompiled.ConnectionData{
		PublicAddress: g.signer.Address(),
		Url:           g.url,
		CreatedAt:     now,
		Digest:        digest[:],
		Signature:     signature,
	})

	for address, nd := range g.nodes {
		data := initConnectionData(address, nd.url, now)
		digest, signature := g.signer.Sign(data)
		connected.Connections = append(connected.Connections, &protobufcompiled.ConnectionData{
			PublicAddress: address,
			Url:           nd.url,
			CreatedAt:     now,
			Digest:        digest[:],
			Signature:     signature,
		})
	}

	return connected, nil
}

func (g *gossiper) LoadDag(_ *emptypb.Empty, stream protobufcompiled.GossipAPI_LoadDagServer) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	chVrx, chErr := g.accounter.StreamDAG(ctx)
	var err error
StreamLoop:
	for {
		select {
		case vrx := <-chVrx:
			if vrx == nil {
				break StreamLoop
			}
			vr := mapAccountantVertexToProtoVertex(vrx)
			err = stream.Send(vr)
			if err != nil {
				break StreamLoop
			}
		case err = <-chErr:
			break StreamLoop
		}
	}
	if err != nil {
		g.log.Error(fmt.Sprintf("GRPC streaming DAG failed: %v", err))
	}
	return err
}

func (g *gossiper) Gossip(ctx context.Context, vg *protobufcompiled.VertexGossip) (*emptypb.Empty, error) {
	if vg == nil {
		return nil, ErrNilVertexGossip
	}
	if vg.Vertex == nil {
		return nil, ErrNilVertex
	}
	err := g.vertexCache.Update(func(txn *badger.Txn) error {
		_, err := txn.Get(vg.Vertex.Hash)
		switch err {
		case nil:
			return ErrVertexInCache
		default:
			switch errors.Is(err, badger.ErrKeyNotFound) {
			case true:
			default:
				return err
			}
		}
		return txn.SetEntry(badger.NewEntry(vg.Vertex.Hash, []byte{}).WithTTL(vertexCacheLongevity))
	})
	if err != nil {
		if errors.Is(err, ErrVertexInCache) || errors.Is(err, badger.ErrConflict) {
			return &emptypb.Empty{}, nil
		}
		return nil, ErrUnexpectedGossipFailure
	}

	g.vertexGossipCh <- vg

	return &emptypb.Empty{}, nil
}

func (g *gossiper) updateDag(ctx context.Context, url string) error {
	nd, err := connectToNode(url)
	if err != nil {
		return err
	}
	defer nd.conn.Close()

	stream, err := nd.client.LoadDag(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}

	chVrx := make(chan *accountant.Vertex, 1000)
	ctxx, cancel := context.WithCancelCause(ctx)
	go g.accounter.LoadDag(ctxx, cancel, chVrx)

	var errx error
StreamRcvLoop:
	for {
		select {
		case <-ctxx.Done():
			if err := ctxx.Err(); err != nil && err != context.Canceled {
				errx = err
			}
			break StreamRcvLoop
		default:
			vg, err := stream.Recv()
			if err != nil || vg == nil {
				errx = err
				break StreamRcvLoop
			}
			vrx := mapProtoVertexToAccountantVertex(vg)
			chVrx <- &vrx
		}
	}
	if errx == nil || errx == io.EOF {
		return nil
	}
	cancel(errx)
	return errx
}

func (g *gossiper) runTimeSortVertexGossip(ctx context.Context, cancel context.CancelFunc) {
	defer close(g.vertexGossipTimeSortedCh)
	t := time.NewTicker(time.Millisecond)
	defer t.Stop()
	vertexes := make([]*protobufcompiled.VertexGossip, 0, vertexGossipChCapacity)
	for {
		select {
		case <-ctx.Done():
			return
		case vrx := <-g.vrxCh:
			if vrx == nil {
				continue
			}
			go func(vrx *accountant.Vertex) {
				vg := mapAccountantVertexToProtoVertex(vrx)
				g.vertexGossipCh <- &protobufcompiled.VertexGossip{
					Vertex:    vg,
					Gossipers: []string{g.signer.Address()},
				}
			}(vrx)
		case vg := <-g.vertexGossipCh:
			if vg == nil {
				continue
			}
			vertexes = append(vertexes, vg)
			slices.SortFunc(vertexes, func(a, b *protobufcompiled.VertexGossip) int {
				if a == nil || b == nil {
					return 0
				}
				if a.Vertex.CreaterdAt == b.Vertex.CreaterdAt {
					return 0
				}
				if a.Vertex.CreaterdAt > b.Vertex.CreaterdAt {
					return 1
				}
				return -1
			})
			select {
			case g.vertexGossipTimeSortedCh <- vertexes[0]:
				vertexes = vertexes[1:]
			default:
			}
		case <-t.C:
			if len(vertexes) == 0 {
				continue
			}
			select {
			case g.vertexGossipTimeSortedCh <- vertexes[0]:
				vertexes = vertexes[1:]
			default:
			}
		}
	}
}

func (g *gossiper) runProcessVertexGossip(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case vg := <-g.vertexGossipTimeSortedCh:
			if vg == nil {
				continue
			}
			set := toSet(vg.Gossipers)
			if _, ok := set[g.signer.Address()]; !ok {
				go g.sendToAccountant(ctx, vg.Vertex)
				vg.Gossipers = append(vg.Gossipers, g.signer.Address())

			}
			vg.Gossipers = toSlice(set)
			g.mux.RLock()
			for addr, nd := range g.nodes {
				if _, ok := set[addr]; ok {
					continue
				}
				go func(client protobufcompiled.GossipAPIClient, addr, url string) {
					if _, err := client.Gossip(ctx, vg); err != nil {
						g.log.Error(
							fmt.Sprintf(
								"Gossiping to [ %s ] with URL [ %s ] vertex %v failed with err: %s",
								addr, url, vg.Vertex.Hash, err),
						)
					}
				}(nd.client, addr, nd.url)
			}
			g.mux.RUnlock()
		}
	}
}

func (g *gossiper) closeAllNodesConnections() {
	for addr, nd := range g.nodes {
		if err := nd.conn.Close(); err != nil {
			g.log.Error(fmt.Sprintf("Closing connection to address [ %s ] with URL [ %s ] failed.", addr, nd.url))
		}
	}
	maps.Clear(g.nodes)
}

func (g *gossiper) updateNodesConnectionsFromGensisNode(ctx context.Context, genesisURL string) error {
	if genesisURL == "" {
		g.log.Info(fmt.Sprintf("Genesis Node URL is not specified. Node [ %s ] runs as Genesis Node.", g.signer.Address()))
		return nil
	}
	opts := grpc.WithTransportCredentials(insecure.NewCredentials()) // TODO: remove when credentials are set
	conn, err := grpc.Dial(genesisURL, opts)
	if err != nil {
		return fmt.Errorf("connection refused: %s", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			g.log.Error(fmt.Sprintf("connection close error: %s", err))
		}
	}()

	client := protobufcompiled.NewGossipAPIClient(conn)
	now := uint64(time.Now().UnixNano())
	data := initConnectionData(g.signer.Address(), g.url, now)
	digest, signature := g.signer.Sign(data)

	cd := &protobufcompiled.ConnectionData{
		PublicAddress: g.signer.Address(),
		Url:           g.url,
		CreatedAt:     now,
		Digest:        digest[:],
		Signature:     signature,
	}

	result, err := client.Discover(ctx, cd)
	if err != nil {
		return fmt.Errorf("result error: %s", err)
	}

	g.mux.Lock()
	defer g.mux.Unlock()

	for _, n := range result.Connections {
		if n.PublicAddress == g.signer.Address() || n.Url == g.url {
			continue
		}
		err := g.valiudateSignature(result.SignerPublicAddress, n.PublicAddress, n.Url, n.CreatedAt, n.Signature, [32]byte(n.Digest))
		if err != nil {
			g.log.Warn(
				fmt.Sprintf("Received connection [ %s ] for URL [ %s ] has corrupted signature. Signer [ %s ], %s.",
					n.PublicAddress, n.Url, result.SignerPublicAddress, err),
			)
			continue
		}
		nd, err := connectToNode(n.Url)
		if err != nil {
			g.log.Error(fmt.Sprintf("Connection to  [ %s ] for URL [ %s ] failed, %s.", n.PublicAddress, n.Url, err))
			continue
		}
		g.nodes[n.PublicAddress] = nd
		g.log.Info(fmt.Sprintf("Node [ %s ] connected to [ %s ] with URL [ %s ].", g.signer.Address(), n.PublicAddress, n.Url))

		if n.Url == genesisURL {
			continue
		}
		if _, err := nd.client.Announce(ctx, cd); err != nil {
			g.log.Info(fmt.Sprintf("Node [ %s ] connection back to [ %s ] with URL [ %s ] failed.", g.signer.Address(), n.PublicAddress, n.Url))
		}
	}

	return nil
}

func (g *gossiper) sendToAccountant(ctx context.Context, vg *protobufcompiled.Vertex) {
	if vg.Transaction == nil {
		return
	}
	v := mapProtoVertexToAccountantVertex(vg)
	if err := g.accounter.AddLeaf(ctx, &v); err != nil {
		g.log.Info(fmt.Sprintf("Node [ %s ] adding leaf error: %s.", g.signer.Address(), err))
	}
}

func (g *gossiper) valiudateSignature(sigAddr, pubAddr, url string, createdAt uint64, signature []byte, hash [32]byte) error {
	data := initConnectionData(pubAddr, url, createdAt)
	return g.verifier.Verify(data, signature, hash, sigAddr)
}

func connectToNode(url string) (nodeData, error) {
	opts := grpc.WithTransportCredentials(insecure.NewCredentials()) // TODO: remove when credentials are set
	conn, err := grpc.Dial(url, opts)
	if err != nil {
		return nodeData{}, fmt.Errorf("dial failed, %s", err)
	}

	client := protobufcompiled.NewGossipAPIClient(conn)
	return nodeData{
		url:    url,
		conn:   conn,
		client: client,
	}, nil
}

func initConnectionData(publicAddress, url string, createdAt uint64) []byte {
	blockData := make([]byte, 0, 8)
	blockData = binary.LittleEndian.AppendUint64(blockData, createdAt)
	return bytes.Join([][]byte{
		[]byte(publicAddress), []byte(url), blockData,
	},
		[]byte{},
	)
}

func toSet(s []string) map[string]struct{} {
	m := make(map[string]struct{}, len(s))
	for _, member := range s {
		m[member] = struct{}{}
	}
	return m
}

func toSlice(m map[string]struct{}) []string {
	s := make([]string, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	return s
}

func mapProtoVertexToAccountantVertex(vg *protobufcompiled.Vertex) accountant.Vertex {
	return accountant.Vertex{
		SignerPublicAddress: vg.SignerPublicAddress,
		CreatedAt:           time.Unix(0, int64(vg.CreaterdAt)),
		Signature:           vg.Signature,
		Transaction: transaction.Transaction{
			CreatedAt:         time.Unix(0, int64(vg.CreaterdAt)),
			IssuerAddress:     vg.Transaction.IssuerAddress,
			ReceiverAddress:   vg.Transaction.ReceiverAddress,
			Subject:           vg.Transaction.Subject,
			Data:              vg.Transaction.Data,
			IssuerSignature:   vg.Transaction.IssuerSignature,
			ReceiverSignature: vg.Transaction.ReceiverSignature,
			Hash:              [32]byte(vg.Transaction.Hash),
			Spice: spice.Melange{
				Currency:              vg.Transaction.Spice.Currency,
				SupplementaryCurrency: vg.Transaction.Spice.SuplementaryCurrency,
			},
		},
		Hash:            [32]byte(vg.Hash),
		LeftParentHash:  [32]byte(vg.LeftParentHash),
		RightParentHash: [32]byte(vg.RightParentHash),
		Weight:          vg.Weight,
	}
}

func mapAccountantVertexToProtoVertex(vrx *accountant.Vertex) *protobufcompiled.Vertex {
	return &protobufcompiled.Vertex{
		SignerPublicAddress: vrx.SignerPublicAddress,
		CreaterdAt:          uint64(vrx.CreatedAt.UnixNano()),
		Signature:           vrx.Signature,
		Transaction: &protobufcompiled.Transaction{
			Subject:           vrx.Transaction.Subject,
			Data:              vrx.Transaction.Data,
			Hash:              vrx.Transaction.Hash[:],
			CreatedAt:         uint64(vrx.Transaction.CreatedAt.UnixNano()),
			ReceiverAddress:   vrx.Transaction.ReceiverAddress,
			IssuerAddress:     vrx.Transaction.IssuerAddress,
			ReceiverSignature: vrx.Transaction.ReceiverSignature,
			IssuerSignature:   vrx.Transaction.IssuerSignature,
			Spice: &protobufcompiled.Spice{
				Currency:             vrx.Transaction.Spice.Currency,
				SuplementaryCurrency: vrx.Transaction.Spice.SupplementaryCurrency,
			},
		},
		Hash:            vrx.Hash[:],
		LeftParentHash:  vrx.LeftParentHash[:],
		RightParentHash: vrx.RightParentHash[:],
		Weight:          vrx.Weight,
	}
}
