package gossip

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/protobufcompiled"
	"github.com/bartossh/Computantis/storage"
	"github.com/bartossh/Computantis/versioning"
	"github.com/dgraph-io/badger/v4"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

type Config struct {
	URL            string `yaml:"url"`
	GenesisURL     string `yaml:"genesis_url"`
	VerticesDBPath string `yaml:"vertices_db_path"`
	Port           int    `yaml:"port"`
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
	return nil
}

type signatureVerifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

type signer interface {
	Sign(message []byte) (digest [32]byte, signature []byte)
	Address() string
}

type gossiper struct {
	protobufcompiled.UnimplementedGossipAPIServer
	verifier                 signatureVerifier
	signer                   signer
	log                      logger.Logger
	vertexCache              *badger.DB
	vertexGossipCh           chan *protobufcompiled.VertexGossip
	vertexGossipTimeSortedCh chan *protobufcompiled.VertexGossip
	nodes                    map[string]nodeData
	url                      string
	mux                      sync.RWMutex
	timeout                  time.Duration
}

// RunGRPC runs the service application that exposes the GRPC API for gossip protocol.
// To stop server cancel the context.
func RunGRPC(ctx context.Context, cfg Config, l logger.Logger, t time.Duration, s signer, v signatureVerifier) error {
	if err := cfg.verify(); err != nil {
		return err
	}

	ctxx, cancel := context.WithCancel(ctx)
	defer cancel()

	db, err := storage.CreateBadgerDB(ctx, cfg.VerticesDBPath, l)
	if err != nil {
		return err
	}

	g := gossiper{
		verifier:                 v,
		signer:                   s,
		log:                      l,
		vertexCache:              db,
		vertexGossipCh:           make(chan *protobufcompiled.VertexGossip, vertexGossipChCapacity),
		vertexGossipTimeSortedCh: make(chan *protobufcompiled.VertexGossip, 1),
		nodes:                    make(map[string]nodeData),
		url:                      cfg.URL,
		mux:                      sync.RWMutex{},
		timeout:                  t,
	}

	defer g.closeAllNodesConnections()
	defer close(g.vertexGossipCh)

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%v", cfg.Port))
	if err != nil {
		cancel()
		return err
	}

	grpcServer := grpc.NewServer(grpc.Creds(credentials.NewClientTLSFromCert(x509.NewCertPool(), "")))
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

	time.Sleep(time.Millisecond * 50) // jsut wait so the server can start

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

func (g *gossiper) Discover(_ context.Context, cd *protobufcompiled.ConnectionData) (*protobufcompiled.ConnectedNodes, error) {
	createdAt := time.Unix(0, int64(cd.CreatedAt))
	err := g.valiudateSignature(cd.PublicAddress, cd.PublicAddress, cd.Url, createdAt, cd.Signature, [32]byte(cd.Digest))
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

	g.nodes[cd.PublicAddress] = nd
	g.log.Info(fmt.Sprintf("Node [ %s ] accepted connection from [ %s ] with URL [ %s ]", g.signer.Address(), cd.PublicAddress, nd.url))

	connected := &protobufcompiled.ConnectedNodes{
		SignerPublicAddress: g.signer.Address(),
		Connections:         make([]*protobufcompiled.ConnectionData, 0, len(g.nodes)+1),
	}
	now := time.Now()
	data := initConnectionData(g.signer.Address(), g.url, now)
	digest, signature := g.signer.Sign(data)

	connected.Connections = append(connected.Connections, &protobufcompiled.ConnectionData{
		PublicAddress: g.signer.Address(),
		Url:           g.url,
		CreatedAt:     uint64(now.Unix()),
		Digest:        digest[:],
		Signature:     signature,
	})

	for address, nd := range g.nodes {
		data = initConnectionData(address, nd.url, now)
		digest, signature = g.signer.Sign(data)
		connected.Connections = append(connected.Connections, &protobufcompiled.ConnectionData{
			PublicAddress: address,
			Url:           nd.url,
			CreatedAt:     uint64(now.Unix()),
			Digest:        digest[:],
			Signature:     signature,
		})
	}

	return connected, nil
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
		if errors.Is(err, ErrVertexInCache) {
			return &emptypb.Empty{}, nil
		}
		return nil, ErrUnexpectedGossipFailure
	}

	g.vertexGossipCh <- vg

	return &emptypb.Empty{}, nil
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
			g.sendToAccountant(ctx, vg.Vertex)
			vg.Gossipers = append(vg.Gossipers, g.signer.Address())
			set := toSet(vg.Gossipers)
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

	client := protobufcompiled.NewGossipAPIClient(conn)
	now := time.Now()
	data := initConnectionData(g.signer.Address(), g.url, now)
	digest, signature := g.signer.Sign(data)

	cd := &protobufcompiled.ConnectionData{
		PublicAddress: g.signer.Address(),
		Url:           g.url,
		CreatedAt:     uint64(now.UnixNano()),
		Digest:        digest[:],
		Signature:     signature,
	}

	result, err := client.Discover(ctx, cd)
	if err != nil {
		return fmt.Errorf("result error: %s", err)
	}
	if err := conn.Close(); err != nil {
		return fmt.Errorf("connection close error: %s", err)
	}

	g.mux.Lock()
	defer g.mux.Unlock()

	for _, n := range result.Connections {
		if n.PublicAddress == g.signer.Address() || n.Url == g.url {
			continue
		}
		createdAt := time.Unix(0, int64(n.CreatedAt))
		err := g.valiudateSignature(result.SignerPublicAddress, n.PublicAddress, n.Url, createdAt, n.Signature, [32]byte(n.Digest))
		if err != nil {
			g.log.Warn(
				fmt.Sprintf("Received connection [ %s ] for URL [ %s ] has corrupted signature. Signer [ %s ]",
					n.PublicAddress, n.Url, result.SignerPublicAddress),
			)
			continue
		}
		nd, err := connectToNode(n.Url)
		if err != nil {
			g.log.Error(fmt.Sprintf("Connection to  [ %s ] for URL [ %s ] failed, %s.", n.PublicAddress, n.Url, err))
			continue
		}
		g.nodes[n.PublicAddress] = nd
		g.log.Info(fmt.Sprintf("Node [ %s ] connected to [ %s ] with URL [ %s ].", g.signer.Address(), n.PublicAddress, nd.url))
	}

	return nil
}

func (g *gossiper) sendToAccountant(ctx context.Context, vg *protobufcompiled.Vertex) {
	// TODO: send to accountant DAG when implementd
	fmt.Printf("unimplementd for vg created at: [ %v ] \n", vg.CreaterdAt)

	// NOTE: after transformation from protobufcompiled.Vertex to accountant.Vertex, accountant can process leaf concurently
}

func (g *gossiper) valiudateSignature(sigAddr, pubAddr, url string, createdAt time.Time, signature []byte, hash [32]byte) error {
	data := initConnectionData(pubAddr, url, createdAt)
	return g.verifier.Verify(data, signature, hash, sigAddr)
}

func connectToNode(url string) (nodeData, error) {
	opts := grpc.WithTransportCredentials(insecure.NewCredentials()) // TODO: remove when credentials are set
	conn, err := grpc.Dial(url, opts)
	if err != nil {
		return nodeData{}, err
	}

	client := protobufcompiled.NewGossipAPIClient(conn)
	return nodeData{
		url:    url,
		conn:   conn,
		client: client,
	}, nil
}

func initConnectionData(publicAddress, url string, createdAt time.Time) []byte {
	blockData := make([]byte, 0, 8)
	blockData = binary.LittleEndian.AppendUint64(blockData, uint64(createdAt.UnixNano()))
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
