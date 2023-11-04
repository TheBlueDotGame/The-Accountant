package gossip

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bartossh/Computantis/accountant"
	"github.com/bartossh/Computantis/logging"
	"github.com/bartossh/Computantis/protobufcompiled"
	"github.com/bartossh/Computantis/stdoutwriter"
	"github.com/bartossh/Computantis/wallet"
	"gotest.tools/v3/assert"
)

func generateData(l int) []byte {
	data := make([]byte, 0, l)
	for i := 0; i < l; i++ {
		data = append(data, byte(rand.Intn(255)))
	}
	return data
}

type discoveryConnetionLogger struct {
	ports   []int
	counter atomic.Int64
}

func newDiscoveryConnetionLogger(ports []int) *discoveryConnetionLogger {
	return &discoveryConnetionLogger{ports: ports, counter: atomic.Int64{}}
}

func (d *discoveryConnetionLogger) Write(p []byte) (n int, err error) {
	for _, port := range d.ports {
		substring := strconv.Itoa(port)
		if strings.Contains(string(p), substring) && strings.Contains(string(p), "connected to") {
			d.counter.Add(1)
		}
	}
	return len(p), nil
}

func (d *discoveryConnetionLogger) readCounter() int64 {
	return d.counter.Load()
}

type testAccountant struct {
	counter    atomic.Uint64
	hasGenesis atomic.Bool
}

func (t *testAccountant) AddLeaf(ctx context.Context, leaf *accountant.Vertex) error {
	t.counter.Add(1)
	return nil
}

func (t *testAccountant) AcceptGenesis(vrx *accountant.Vertex) error {
	t.counter.Add(1)
	t.hasGenesis.Store(true)
	return nil
}

func (t *testAccountant) StreamDAG(ctx context.Context) (<-chan *accountant.Vertex, <-chan error) {
	return nil, nil
}

func (t *testAccountant) LoadDag(ctx context.Context, cancelF context.CancelCauseFunc, cVrx <-chan *accountant.Vertex) {
	return
}

func (t *testAccountant) DagLoaded() bool {
	return true
}

func (t *testAccountant) readCounter() uint64 {
	return t.counter.Load()
}

func TestDiscoverProtocol(t *testing.T) {
	testsCases := []struct {
		nodes      []int
		handshakes int
	}{
		{handshakes: 2, nodes: []int{8080, 8081}},
		{handshakes: 6, nodes: []int{8080, 8081, 8082}},
		{handshakes: 12, nodes: []int{8080, 8081, 8082, 8083}},
		{handshakes: 20, nodes: []int{8080, 8081, 8082, 8083, 8084}},
		{handshakes: 30, nodes: []int{8080, 8081, 8082, 8083, 8084, 8085}},
		{handshakes: 42, nodes: []int{8080, 8081, 8082, 8083, 8084, 8085, 8086}},
		{handshakes: 56, nodes: []int{8080, 8081, 8082, 8083, 8084, 8085, 8086, 8087}},
	}

	for _, c := range testsCases {
		t.Run(fmt.Sprintf("handshakes %v test", c.handshakes), func(t *testing.T) {
			callOnLogErr := func(err error) {
				fmt.Printf("logger failed with error: %s\n", err)
			}
			callOnFail := func(err error) {
				fmt.Printf("Faield with error: %s\n", err)
			}

			counter := newDiscoveryConnetionLogger(c.nodes)

			l := logging.New(callOnLogErr, callOnFail, counter)

			w, err := wallet.New()
			assert.NilError(t, err)

			v := wallet.NewVerifier()

			ctx, cancel := context.WithCancel(context.Background())

			genessisConfig := Config{
				URL:        fmt.Sprintf("localhost:%v", c.nodes[0]),
				GenesisURL: "",
				Port:       c.nodes[0],
			}
			go func() {
				acc := testAccountant{}
				err := RunGRPC(ctx, genessisConfig, l, time.Second*1, &w, v, &acc)
				assert.NilError(t, err)
			}()

			for _, port := range c.nodes[1:] {
				cfg := Config{
					URL:        fmt.Sprintf("localhost:%v", port),
					GenesisURL: fmt.Sprintf("localhost:%v", c.nodes[0]),
					Port:       port,
				}
				go func(cfg Config) {
					acc := testAccountant{}
					w, err := wallet.New()
					assert.NilError(t, err)
					v := wallet.NewVerifier()
					err = RunGRPC(ctx, cfg, l, time.Second*1, &w, v, &acc)
					assert.NilError(t, err)
				}(cfg)
			}

			time.Sleep(time.Second * 1)
			cancel()

			cnt := counter.readCounter()
			fmt.Printf("counter: %v\n", cnt)
			assert.Equal(t, int(cnt), c.handshakes)

			time.Sleep(time.Millisecond * 200)
		})
	}
}

func TestGossipProtocol(t *testing.T) {
	testsCases := []struct {
		nodes []int
	}{
		{nodes: []int{8080, 8081}},
		{nodes: []int{8080, 8081, 8082}},
		{nodes: []int{8080, 8081, 8082, 8083}},
		{nodes: []int{8080, 8081, 8082, 8083, 8084}},
		{nodes: []int{8080, 8081, 8082, 8083, 8084, 8085}},
		{nodes: []int{8080, 8081, 8082, 8083, 8084, 8085, 8086}},
		{nodes: []int{8080, 8081, 8082, 8083, 8084, 8085, 8086, 8087}},
	}

	vertexRoundsPerNode := 10

	for _, c := range testsCases {
		t.Run(fmt.Sprintf("gossip %v nodes", len(c.nodes)), func(t *testing.T) {
			callOnLogErr := func(err error) {
				fmt.Printf("logger failed with error: %s\n", err)
			}
			callOnFail := func(err error) {
				fmt.Printf("Faield with error: %s\n", err)
			}

			l := logging.New(callOnLogErr, callOnFail, &stdoutwriter.Logger{})

			w, err := wallet.New()
			assert.NilError(t, err)

			v := wallet.NewVerifier()

			ctx, cancel := context.WithCancel(context.Background())

			genessisConfig := Config{
				URL:        fmt.Sprintf("localhost:%v", c.nodes[0]),
				GenesisURL: "",
				Port:       c.nodes[0],
			}
			go func() {
				acc := testAccountant{}
				err := RunGRPC(ctx, genessisConfig, l, time.Second*1, &w, v, &acc)
				assert.NilError(t, err)
				assert.Equal(t, acc.readCounter(), uint64(len(c.nodes)*vertexRoundsPerNode))
			}()

			var wg sync.WaitGroup
			for _, port := range c.nodes[1:] {
				wg.Add(1)
				cfg := Config{
					URL:        fmt.Sprintf("localhost:%v", port),
					GenesisURL: fmt.Sprintf("localhost:%v", c.nodes[0]),
					Port:       port,
				}
				go func(cfg Config) {
					acc := testAccountant{}
					w, err := wallet.New()
					assert.NilError(t, err)
					v := wallet.NewVerifier()
					go func() {
						time.Sleep(time.Second)
						wg.Done()
					}()
					err = RunGRPC(ctx, cfg, l, time.Second*1, &w, v, &acc)
					assert.NilError(t, err)                                                      // if fails it means nodes are overloded or are not able to handle connections.
					assert.Equal(t, acc.readCounter(), uint64(len(c.nodes)*vertexRoundsPerNode)) // NOTE: The assertion for test of gossip protoco happens here.
					// NOTE: we want to each node to receive exactly the amount of propagated certexes per each node.
				}(cfg)
			}
			wg.Wait()

			for _, port := range c.nodes {
				wg.Add(1)
				go func(port int) {
					nd, err := connectToNode(fmt.Sprintf("localhost:%v", port))
					assert.NilError(t, err)
					for i := 0; i < vertexRoundsPerNode; i++ {
						time.Sleep(time.Millisecond)
						vd := protobufcompiled.VertexGossip{
							Vertex: &protobufcompiled.Vertex{
								Hash:       generateData(32), // TODO: generate real hash and Trx data when accountant is implemented
								CreaterdAt: uint64(time.Now().UnixNano()),
								Transaction: &protobufcompiled.Transaction{
									Hash: generateData(32),
								},
								LeftParentHash:  generateData(32),
								RightParentHash: generateData(32),
							},
							Gossipers: []string{},
						}
						_, err := nd.client.Gossip(ctx, &vd)
						assert.NilError(t, err)
					}
					nd.conn.Close()
					wg.Done()
				}(port)
			}

			wg.Wait()
			time.Sleep(time.Second * 2) // Allow all the nodes inner process to finish.
			cancel()
			time.Sleep(time.Second)
		})
	}
}

//func TestDAGWithGossip(t *testing.T) {
//	testsCases := []struct {
//		nodes []int
//	}{
//		{nodes: []int{8080, 8081}},
//		{nodes: []int{8080, 8081, 8082}},
//		{nodes: []int{8080, 8081, 8082, 8083}},
//		{nodes: []int{8080, 8081, 8082, 8083, 8084}},
//		{nodes: []int{8080, 8081, 8082, 8083, 8084, 8085}},
//		{nodes: []int{8080, 8081, 8082, 8083, 8084, 8085, 8086}},
//		{nodes: []int{8080, 8081, 8082, 8083, 8084, 8085, 8086, 8087}},
//	}
//
//	vertexRoundsPerNode := 10
//
//	for _, c := range testsCases {
//		t.Run(fmt.Sprintf("gossip %v nodes", len(c.nodes)), func(t *testing.T) {
//			callOnLogErr := func(err error) {
//				fmt.Printf("logger failed with error: %s\n", err)
//			}
//			callOnFail := func(err error) {
//				fmt.Printf("Faield with error: %s\n", err)
//			}
//
//			l := logging.New(callOnLogErr, callOnFail, &stdoutwriter.Logger{})
//
//			w, err := wallet.New()
//			assert.NilError(t, err)
//
//			v := wallet.NewVerifier()
//
//			ctx, cancel := context.WithCancel(context.Background())
//
//			genessisConfigNode := Config{
//				URL:        fmt.Sprintf("localhost:%v", c.nodes[0]),
//				GenesisURL: "",
//				Port:       c.nodes[0],
//			}
//			genessisConfigAccountant := accountant.Config{}
//
//			nodesReadyCh := make(chan struct{})
//
//			genessisReceiver, err := wallet.New()
//			assert.NilError(t, err)
//
//			go func() {
//				acc, err := accountant.NewAccountingBook(ctx, genessisConfigAccountant, v, &w, l)
//				assert.NilError(t, err)
//				go func(nodesReadyCh <-chan struct{}) {
//					<-nodesReadyCh
//					acc.CreateGenesis("Genesis Test Transaction", spice.New(1000000000000000, 0), []byte{}, &genessisReceiver)
//				}(nodesReadyCh)
//				err = RunGRPC(ctx, genessisConfigNode, l, time.Second*1, &w, v, acc)
//				assert.NilError(t, err)
//			}()
//
//			var wg sync.WaitGroup
//			for _, port := range c.nodes[1:] {
//				wg.Add(1)
//				cfg := Config{
//					URL:        fmt.Sprintf("localhost:%v", port),
//					GenesisURL: fmt.Sprintf("localhost:%v", c.nodes[0]),
//					Port:       port,
//				}
//				w, err := wallet.New()
//				assert.NilError(t, err)
//				acc, err := accountant.NewAccountingBook(ctx, genessisConfigAccountant, v, &w, l)
//				assert.NilError(t, err)
//				go func(cfg Config) {
//					v := wallet.NewVerifier()
//					go func() {
//						time.Sleep(time.Second)
//						wg.Done()
//					}()
//					err = RunGRPC(ctx, cfg, l, time.Second*1, &w, v, acc)
//					assert.NilError(t, err)
//				}(cfg)
//			}
//			wg.Wait()
//			close(nodesReadyCh)
//
//			for _, port := range c.nodes {
//				wg.Add(1)
//				go func(port int) {
//					nd, err := connectToNode(fmt.Sprintf("localhost:%v", port))
//					assert.NilError(t, err)
//					for i := 0; i < vertexRoundsPerNode; i++ {
//						time.Sleep(time.Millisecond)
//						_, err := nd.client.Gossip(ctx, &vd)
//						assert.NilError(t, err)
//
//					}
//					nd.conn.Close()
//					wg.Done()
//				}(port)
//			}
//
//			wg.Wait()
//			time.Sleep(time.Second * 2) // Allow all the nodes inner process to finish.
//			cancel()
//			time.Sleep(time.Second)
//		})
//	}
//}
