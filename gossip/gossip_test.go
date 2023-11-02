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
	fmt.Printf("%s\n", string(p))
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
	nodePorts := []int{8080, 8081, 8082, 8083, 8084, 8085}

	callOnLogErr := func(err error) {
		fmt.Printf("logger failed with error: %s\n", err)
	}
	callOnFail := func(err error) {
		fmt.Printf("Faield with error: %s\n", err)
	}

	counter := newDiscoveryConnetionLogger(nodePorts)

	l := logging.New(callOnLogErr, callOnFail, counter)

	w, err := wallet.New()
	assert.NilError(t, err)

	v := wallet.NewVerifier()

	ctx, cancel := context.WithCancel(context.Background())

	genessisConfig := Config{
		URL:        fmt.Sprintf("localhost:%v", nodePorts[0]),
		GenesisURL: "",
		Port:       nodePorts[0],
	}
	go func() {
		err := RunGRPC(ctx, genessisConfig, l, time.Second*1, &w, v)
		assert.NilError(t, err)
	}()

	for _, port := range nodePorts[1:] {
		cfg := Config{
			URL:        fmt.Sprintf("localhost:%v", port),
			GenesisURL: fmt.Sprintf("localhost:%v", nodePorts[0]),
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

	time.Sleep(time.Second * 2)
	cancel()

	cnt := counter.readCounter()
	fmt.Printf("counter: %v\n", cnt)

	time.Sleep(time.Second * 1)
}
