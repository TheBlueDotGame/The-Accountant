package gossip

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bartossh/Computantis/logging"
	"github.com/bartossh/Computantis/wallet"
	"gotest.tools/v3/assert"
)

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
				err := RunGRPC(ctx, genessisConfig, l, time.Second*1, &w, v)
				assert.NilError(t, err)
			}()

			for _, port := range c.nodes[1:] {
				cfg := Config{
					URL:        fmt.Sprintf("localhost:%v", port),
					GenesisURL: fmt.Sprintf("localhost:%v", c.nodes[0]),
					Port:       port,
				}
				go func(cfg Config) {
					w, err := wallet.New()
					assert.NilError(t, err)
					v := wallet.NewVerifier()
					err = RunGRPC(ctx, cfg, l, time.Second*1, &w, v)
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
