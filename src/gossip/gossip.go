package gossip

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/bartossh/Computantis/src/accountant"
	"github.com/bartossh/Computantis/src/logger"
	"github.com/bartossh/Computantis/src/protobufcompiled"
	"github.com/bartossh/Computantis/src/providers"
	"github.com/bartossh/Computantis/src/spice"
	"github.com/bartossh/Computantis/src/transaction"
	"github.com/bartossh/Computantis/src/transformers"
	"github.com/bartossh/Computantis/src/versioning"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	ErrDiscoveryAttmeptSignatureFailed = errors.New("discovery attempt failed, signature or digest is invalid")
	ErrUnexpectedGossipFailure         = errors.New("unexpected gossip failure")
	ErrVertexInCache                   = errors.New("vertex in cache")
	ErrFailedToProcessGossip           = errors.New("failed to process gossip")
	ErrNilVertex                       = errors.New("vertex is nil")
	ErrNilVrxMsgGossip                 = errors.New("vertex gossip is nil")
	ErrNilTrx                          = errors.New("transaction is nil")
	ErrNilTrxMsgGossip                 = errors.New("transaction gossip is nil")
	ErrNilSignedHash                   = errors.New("signed hash is nil")
)

const (
	vertexCacheLongevity = time.Second * 10
)

const (
	vertexGossipChCapacity = 100
	trxGossipChCappacity   = 100
	rejectHashChCapacity   = 80
	totalRetries           = 10
)

type nodeData struct {
	conn   *grpc.ClientConn
	client protobufcompiled.GossipAPIClient
	url    string
}

// Config is a configuration for the gossip node.
type Config struct {
	URL              string        `yaml:"url"`
	GenesisURL       string        `yaml:"genesis_url"`
	LoadDagURL       string        `yaml:"load_dag_url"`
	GenessisReceiver string        `yaml:"genesis_receiver"`
	GenesisSpice     spice.Melange `yaml:"genesis_spice"`
	Port             int           `yaml:"port"`
}

func (c Config) verify() error {
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("allowed port range is 0 to 65535, got %v", c.Port)
	}
	return nil
}

type signatureVerifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

type accounter interface {
	CreateGenesis(subject string, spc spice.Melange, data []byte, publicAddress string) (accountant.Vertex, error)
	AddLeaf(ctx context.Context, leaf *accountant.Vertex) error
	StreamDAG(ctx context.Context) <-chan *accountant.Vertex
	LoadDag(cancelF context.CancelCauseFunc, cVrx <-chan *accountant.Vertex)
	DagLoaded() bool
}

type piper interface {
	SubscribeToTrx() <-chan *protobufcompiled.Transaction
	SubscribeToVrx() <-chan *accountant.Vertex
}

type gossiper struct {
	protobufcompiled.UnimplementedGossipAPIServer
	accounter accounter
	verifier  signatureVerifier
	signer    accountant.Signer
	log       logger.Logger
	trxCache  providers.AwaitedTrxCacheProviderBalanceCacher
	flash     providers.FlashbackMemoryHashProviderAddressRemover
	piper     piper
	nodes     map[string]nodeData
	url       string
	mux       sync.RWMutex
	timeout   time.Duration
}

// RunGRPC runs the service application that exposes the GRPC API for gossip protocol.
// To stop server cancel the context.
func RunGRPC(ctx context.Context, cfg Config, l logger.Logger, t time.Duration, s accountant.Signer,
	v signatureVerifier, a accounter, trxCache providers.AwaitedTrxCacheProviderBalanceCacher,
	flash providers.FlashbackMemoryHashProviderAddressRemover, p piper,
) error {
	if err := cfg.verify(); err != nil {
		return err
	}

	ctxx, cancel := context.WithCancel(ctx)
	defer cancel()

	g := gossiper{
		accounter: a,
		verifier:  v,
		signer:    s,
		log:       l,
		trxCache:  trxCache,
		flash:     flash,
		piper:     p,
		nodes:     make(map[string]nodeData),
		url:       cfg.URL,
		mux:       sync.RWMutex{},
		timeout:   t,
	}

	switch cfg.LoadDagURL {
	case "":
		if _, err := g.accounter.CreateGenesis(
			"Genesis Vertex",
			spice.New(cfg.GenesisSpice.Currency, cfg.GenesisSpice.SupplementaryCurrency),
			[]byte{}, cfg.GenessisReceiver,
		); err != nil {
			g.log.Error(fmt.Sprintf("failed creating genesis vertex: %s", err))
			return err
		}
	default:
		if err := g.updateDag(ctx, cfg.LoadDagURL); err != nil {
			g.log.Error(fmt.Sprintf("failed loading DAG: %s", err))
			return err
		}
		g.log.Info(fmt.Sprintf("node %s loaded DAG from URL: %s.", g.signer.Address(), cfg.URL))
	}

	defer g.closeAllNodesConnections()

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%v", cfg.Port))
	if err != nil {
		cancel()
		return err
	}

	grpcServer := grpc.NewServer()
	protobufcompiled.RegisterGossipAPIServer(grpcServer, &g)

	go g.runTransactionGossipProcess(ctx)
	go g.runVertexGossipProcess(ctx)

	go func() {
		err = grpcServer.Serve(lis)
		if err != nil {
			g.log.Fatal(fmt.Sprintf("node [ %s ] cannot start server on port [ %v ], %s", s.Address(), cfg.Port, err))
			cancel()
		}
	}()

	time.Sleep(time.Millisecond * 50) // just wait so the server can start

	defer grpcServer.GracefulStop()

	if err == nil {
		if err := g.updateNodesConnectionsFromGensisNode(ctx, cfg.GenesisURL); err != nil {
			g.log.Fatal(
				fmt.Sprintf("updating nodes on genesis URL [ %s ] for node [ %s ] failed, %s",
					cfg.GenesisURL, g.signer.Address(), err.Error(),
				),
			)
			cancel()
		}
	}

	g.log.Info(fmt.Sprintf("server started on port [ %v ] for node [ %s ].", cfg.Port, s.Address()))

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
	err := g.validateSignature(cd.PublicAddress, cd.PublicAddress, cd.Url, cd.CreatedAt, cd.Signature, [32]byte(cd.Digest))
	if err != nil {
		g.log.Info(fmt.Sprintf("discovery attempt failed, public address [ %s ] with URL [ %s ], %s", cd.PublicAddress, cd.Url, err))
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
	g.log.Info(fmt.Sprintf("node [ %s ] connected to [ %s ] with URL [ %s ].", g.signer.Address(), cd.PublicAddress, cd.Url))

	return &emptypb.Empty{}, nil
}

func (g *gossiper) Discover(_ context.Context, cd *protobufcompiled.ConnectionData) (*protobufcompiled.ConnectedNodes, error) {
	err := g.validateSignature(cd.PublicAddress, cd.PublicAddress, cd.Url, cd.CreatedAt, cd.Signature, [32]byte(cd.Digest))
	if err != nil {
		g.log.Info(fmt.Sprintf("discovery attempt failed, public address [ %s ] with URL [ %s ], %s", cd.PublicAddress, cd.Url, err))
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
	g.log.Info(fmt.Sprintf("node [ %s ] connected to [ %s ] with URL [ %s ].", g.signer.Address(), cd.PublicAddress, cd.Url))

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
	chVrx := g.accounter.StreamDAG(ctx)
	var err error
streamLoop:
	for vrx := range chVrx {
		if vrx == nil {
			break streamLoop
		}
		vr := mapAccountantVertexToProtoVertex(vrx)
		var retriesCount int
		var isSend bool
	sendLoop:
		for !isSend && retriesCount < totalRetries {
			err = stream.Send(vr)
			if err != nil {
				retriesCount++
				continue sendLoop
			}
			isSend = true
		}
		if !isSend {
			break streamLoop
		}
	}
	if err != nil {
		g.log.Error(fmt.Sprintf("GRPC streaming DAG failed: %v", err))
	}
	return err
}

func (g *gossiper) GossipVrx(ctx context.Context, vg *protobufcompiled.VrxMsgGossip) (*emptypb.Empty, error) {
	if vg == nil {
		return nil, ErrNilVrxMsgGossip
	}
	if vg.Vertex == nil {
		return nil, ErrNilVertex
	}
	ok, err := g.flash.HasHash(vg.Vertex.Hash)
	if err != nil {
		g.log.Error(fmt.Sprintf("gossip vertex failed reading flashback memory about hash [ %x ], %s", vg.Vertex.Hash, err))
	}
	if ok {
		return &emptypb.Empty{}, nil
	}

	set := g.verifyGossipers([32]byte(vg.Vertex.Hash), vg.Gossipers)
	if _, ok := set[g.signer.Address()]; !ok {
		if err := g.sendToAccountant(ctx, vg.Vertex); err != nil {
			g.log.Info(fmt.Sprintf("node [ %s ] adding leaf error: %s.", g.signer.Address(), err))
			return nil, ErrFailedToProcessGossip
		}
		_, err := g.trxCache.RemoveAwaitedTransaction([32]byte(vg.Vertex.Transaction.Hash), vg.Vertex.Transaction.ReceiverAddress)
		if err != nil {
			g.log.Info(fmt.Sprintf("node [ %s ] removing trx %v from cache error: %s.", g.signer.Address(), vg.Vertex.Transaction.Hash, err))
		}
		digest, signature := g.signer.Sign(createGossiperMessageToSign(g.signer.Address(), [32]byte(vg.Vertex.Hash)))
		set[g.signer.Address()] = &protobufcompiled.Gossiper{
			Address:   g.signer.Address(),
			Digest:    digest[:],
			Signature: signature,
		}
		vg.Gossipers = toSlice(set)
		g.gossipVertex(ctx, vg, set)
	}

	go func() {
		if err := g.flash.RemoveAddress(vg.Vertex.Transaction.IssuerAddress); err != nil {
			g.log.Error(fmt.Sprintf("confirm endpoint, removing issuer address [ %s ] from flash failed: %s", vg.Vertex.Transaction.IssuerAddress, err))
		}
		if err := g.flash.RemoveAddress(vg.Vertex.Transaction.ReceiverAddress); err != nil {
			g.log.Error(fmt.Sprintf("confirm endpoint, removing receiver address [ %s ] from flash failed: %s", vg.Vertex.Transaction.ReceiverAddress, err))
		}
		if err := g.trxCache.RemoveBalance(vg.Vertex.Transaction.IssuerAddress); err != nil {
			g.log.Error(fmt.Sprintf("confirm endpoint, removing cached balance for address [ %s ] from flash failed: %s", vg.Vertex.Transaction.IssuerAddress, err))
		}
		if err := g.trxCache.RemoveBalance(vg.Vertex.Transaction.ReceiverAddress); err != nil {
			g.log.Error(fmt.Sprintf("confirm endpoint, removing cached balance for address [ %s ] from flash failed: %s", vg.Vertex.Transaction.ReceiverAddress, err))
		}
	}()

	return &emptypb.Empty{}, nil
}

func (g *gossiper) GossipTrx(ctx context.Context, tg *protobufcompiled.TrxMsgGossip) (*emptypb.Empty, error) {
	if tg == nil {
		return nil, ErrNilTrxMsgGossip
	}
	if tg.Trx == nil {
		return nil, ErrNilTrx
	}

	ok, err := g.flash.HasHash(tg.Trx.Hash)
	if err != nil {
		g.log.Error(fmt.Sprintf("gossip trx failed reading flashback memory about hash [ %x ], %s", tg.Trx.Hash, err))
	}
	if ok {
		return &emptypb.Empty{}, nil
	}

	set := g.verifyGossipers([32]byte(tg.Trx.Hash), tg.Gossipers)
	if _, ok := set[g.signer.Address()]; !ok {
		trx, err := transformers.ProtoTrxToTrx(tg.Trx)
		if err != nil {
			g.log.Error(fmt.Sprintf("transaction gossiper trx %v transformation failed, %s", trx.Hash, err))
			return nil, ErrFailedToProcessGossip
		}
		if err := trx.VerifyIssuer(g.verifier); err != nil {
			g.log.Error(fmt.Sprintf("transaction gossiper trx %v verification failed, %s", trx.Hash, err))
			return nil, ErrFailedToProcessGossip
		}
		if err := g.trxCache.SaveAwaitedTransaction(&trx); err != nil {
			g.log.Error(fmt.Sprintf("transaction gossiper trx %v saving failed, %s", trx.Hash, err))
		}
		digest, signature := g.signer.Sign(createGossiperMessageToSign(g.signer.Address(), [32]byte(tg.Trx.Hash)))
		set[g.signer.Address()] = &protobufcompiled.Gossiper{
			Address:   g.signer.Address(),
			Digest:    digest[:],
			Signature: signature,
		}
		tg.Gossipers = toSlice(set)
		g.gossipTransaction(ctx, tg, set)
	}
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
	go g.accounter.LoadDag(cancel, chVrx)

	var errx error
	defer close(chVrx)
	defer cancel(errx)
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
	return errx
}

func (g *gossiper) runVertexGossipProcess(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case vrx := <-g.piper.SubscribeToVrx():
			if vrx == nil {
				continue
			}
			vp := mapAccountantVertexToProtoVertex(vrx)
			digest, signature := g.signer.Sign(createGossiperMessageToSign(g.signer.Address(), vrx.Hash))
			gossiper := &protobufcompiled.Gossiper{
				Address:   g.signer.Address(),
				Digest:    digest[:],
				Signature: signature,
			}
			vg := &protobufcompiled.VrxMsgGossip{
				Vertex:    vp,
				Gossipers: []*protobufcompiled.Gossiper{gossiper},
			}
			set := map[string]*protobufcompiled.Gossiper{g.signer.Address(): gossiper}
			g.gossipVertex(ctx, vg, set)
		}
	}
}

func (g *gossiper) runTransactionGossipProcess(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case tx := <-g.piper.SubscribeToTrx():
			if tx == nil {
				continue
			}
			digest, signature := g.signer.Sign(createGossiperMessageToSign(g.signer.Address(), [32]byte(tx.Hash)))
			gossiper := &protobufcompiled.Gossiper{
				Address:   g.signer.Address(),
				Digest:    digest[:],
				Signature: signature,
			}
			tg := &protobufcompiled.TrxMsgGossip{
				Trx:       tx,
				Gossipers: []*protobufcompiled.Gossiper{gossiper},
			}
			set := map[string]*protobufcompiled.Gossiper{g.signer.Address(): gossiper}
			g.gossipTransaction(ctx, tg, set)
		}
	}
}

func (g *gossiper) gossipVertex(ctx context.Context, vg *protobufcompiled.VrxMsgGossip, set map[string]*protobufcompiled.Gossiper) {
	g.mux.RLock()
	defer g.mux.RUnlock()
	for addr, nd := range g.nodes {
		if _, ok := set[addr]; ok {
			continue
		}
		go func(client protobufcompiled.GossipAPIClient, addr, url string) {
			if _, err := client.GossipVrx(ctx, vg); err != nil {
				g.log.Error(
					fmt.Sprintf(
						"gossiping to [ %s ] with URL [ %s ] vertex %v failed with err: %s",
						addr, url, vg.Vertex.Hash, err),
				)
			}
		}(nd.client, addr, nd.url)
	}
}

func (g *gossiper) gossipTransaction(ctx context.Context, tg *protobufcompiled.TrxMsgGossip, set map[string]*protobufcompiled.Gossiper) {
	g.mux.RLock()
	defer g.mux.RUnlock()
	for addr, nd := range g.nodes {
		if _, ok := set[addr]; ok {
			continue
		}
		go func(client protobufcompiled.GossipAPIClient, addr, url string) {
			if _, err := client.GossipTrx(ctx, tg); err != nil {
				g.log.Error(
					fmt.Sprintf(
						"gossiping to [ %s ] with URL [ %s ] transaction %v failed with err: %s",
						addr, url, tg.Trx.Hash, err),
				)
			}
		}(nd.client, addr, nd.url)
	}
}

func (g *gossiper) closeAllNodesConnections() {
	for addr, nd := range g.nodes {
		if err := nd.conn.Close(); err != nil {
			g.log.Error(fmt.Sprintf("closing connection to address [ %s ] with URL [ %s ] failed.", addr, nd.url))
		}
	}
	maps.Clear(g.nodes)
}

func (g *gossiper) updateNodesConnectionsFromGensisNode(ctx context.Context, genesisURL string) error {
	if genesisURL == "" {
		g.log.Info(fmt.Sprintf("genesis Node URL is not specified. Node [ %s ] runs as Genesis Node.", g.signer.Address()))
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
		err := g.validateSignature(result.SignerPublicAddress, n.PublicAddress, n.Url, n.CreatedAt, n.Signature, [32]byte(n.Digest))
		if err != nil {
			g.log.Warn(
				fmt.Sprintf("received connection [ %s ] for URL [ %s ] has corrupted signature. Signer [ %s ], %s.",
					n.PublicAddress, n.Url, result.SignerPublicAddress, err),
			)
			continue
		}
		nd, err := connectToNode(n.Url)
		if err != nil {
			g.log.Error(fmt.Sprintf("connection to  [ %s ] for URL [ %s ] failed, %s.", n.PublicAddress, n.Url, err))
			continue
		}
		g.nodes[n.PublicAddress] = nd
		g.log.Info(fmt.Sprintf("node [ %s ] connected to [ %s ] with URL [ %s ].", g.signer.Address(), n.PublicAddress, n.Url))

		if n.Url == genesisURL {
			continue
		}
		if _, err := nd.client.Announce(ctx, cd); err != nil {
			g.log.Info(fmt.Sprintf("node [ %s ] connection back to [ %s ] with URL [ %s ] failed.", g.signer.Address(), n.PublicAddress, n.Url))
		}
	}

	return nil
}

func (g *gossiper) sendToAccountant(ctx context.Context, vg *protobufcompiled.Vertex) error {
	if vg.Transaction == nil {
		return ErrNilTrx
	}
	v := mapProtoVertexToAccountantVertex(vg)
	if err := g.accounter.AddLeaf(ctx, &v); err != nil {
		return err
	}
	return nil
}

func (g *gossiper) validateSignature(sigAddr, pubAddr, url string, createdAt uint64, signature []byte, hash [32]byte) error {
	data := initConnectionData(pubAddr, url, createdAt)
	return g.verifier.Verify(data, signature, hash, sigAddr)
}

func (g *gossiper) verifyGossipers(hash [32]byte, s []*protobufcompiled.Gossiper) map[string]*protobufcompiled.Gossiper {
	m := make(map[string]*protobufcompiled.Gossiper, len(s))
	for _, member := range s {
		err := g.verifier.Verify(createGossiperMessageToSign(member.Address, hash), member.Signature, [32]byte(member.Digest), member.Address)
		if err != nil {
			g.log.Error(fmt.Sprintf("verifying gossiper address [ %s ] invalid signature for hash %v gossip", member.Address, hash))
			continue
		}
		m[member.Address] = member
	}
	return m
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

func toSlice(m map[string]*protobufcompiled.Gossiper) []*protobufcompiled.Gossiper {
	s := make([]*protobufcompiled.Gossiper, 0, len(m))
	for _, v := range m {
		s = append(s, v)
	}
	return s
}

func mapProtoVertexToAccountantVertex(vg *protobufcompiled.Vertex) accountant.Vertex {
	return accountant.Vertex{
		SignerPublicAddress: vg.SignerPublicAddress,
		CreatedAt:           time.Unix(0, int64(vg.CreaterdAt)),
		Signature:           vg.Signature,
		Transaction: transaction.Transaction{
			CreatedAt:         time.Unix(0, int64(vg.Transaction.CreatedAt)),
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

func createGossiperMessageToSign(address string, hash [32]byte) []byte {
	return append([]byte(address), hash[:]...)
}
