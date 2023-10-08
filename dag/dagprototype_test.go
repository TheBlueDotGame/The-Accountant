//go:build prototype

package dag

import (
	"fmt"
	"testing"
	"time"

	"github.com/inancgumus/screen"
)

func TestPrototype(t *testing.T) {
	probes := 100
	sleep := time.Millisecond * 50

	screen.Clear()
	fmt.Printf("Starting test for [ %v ] probes and sleep time [ %v ]\n\n", probes, sleep)

	for i := 0; i < probes; i++ {
		screen.Clear()
		screen.MoveTopLeft()
		runGoshipPrototypeTest(true)
		fmt.Printf("\n\nProbe [ %v ] done.\n\n", i+1)
		time.Sleep(sleep)
	}
	time.Sleep(time.Second)
	screen.Clear()
	screen.MoveTopLeft()
	fmt.Printf("All of [ %v ] probes done./n", probes+1)
}

func BenchmarkPrototype(b *testing.B) {
	for n := 0; n < b.N; n++ {
		runGoshipPrototypeTest(false)
	}
}
