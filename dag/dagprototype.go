package dag

import (
	"context"
	"fmt"

	"golang.org/x/exp/maps"
)

type message struct {
	knowledgeable map[*node]struct{}
	v             int
}

type node struct {
	messages  map[int]struct{}
	nodes     map[*node]struct{}
	messageCh chan message
}

func newNode() *node {
	return &node{
		messages:  make(map[int]struct{}),
		nodes:     make(map[*node]struct{}),
		messageCh: make(chan message, 10000),
	}
}

func (n *node) run(ctx context.Context, collectedNum int) <-chan struct{} {
	ch := make(chan struct{}, 1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case v := <-n.messageCh:
				if v.knowledgeable == nil {
					v.knowledgeable = make(map[*node]struct{})
				}
				if _, ok := v.knowledgeable[n]; ok {
					continue
				}
				if _, ok := n.messages[v.v]; !ok {
					v.knowledgeable[n] = struct{}{}
					n.messages[v.v] = struct{}{}
					for cn := range n.nodes {
						if _, ok := v.knowledgeable[cn]; ok {
							continue
						}
						nv := message{
							v:             v.v,
							knowledgeable: maps.Clone(v.knowledgeable),
						}
						cn.receive(nv)
					}
					collectedNum--
					if collectedNum == 0 {
						close(ch)
					}
				}
			}
		}
	}()
	return ch
}

func (n *node) connect(nn *node) {
	n.nodes[nn] = struct{}{}
}

func (n *node) receive(m message) {
	n.messageCh <- m
}

func runGoshipPrototypeTest(print bool) {
	nodesCases := []struct {
		n      *node
		values []int
	}{
		{
			n:      newNode(),
			values: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
		{
			n:      newNode(),
			values: []int{10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
		},
		{
			n:      newNode(),
			values: []int{20, 21, 22, 23, 24, 25, 26, 27, 28, 29},
		},
		{
			n:      newNode(),
			values: []int{30, 31, 32, 33, 34, 35, 36, 37, 38, 39},
		},
		{
			n:      newNode(),
			values: []int{40, 41, 42, 43, 44, 45, 46, 47, 48, 49},
		},
		{
			n:      newNode(),
			values: []int{50, 51, 52, 53, 54, 55, 56, 57, 58, 59},
		},
	}
	upperValuesRange := 59

	ctx, cancel := context.WithCancel(context.Background())

	semaphores := make([]<-chan struct{}, 0, len(nodesCases))

	for _, nc0 := range nodesCases {
		semaphore := nc0.n.run(ctx, upperValuesRange+1)
		semaphores = append(semaphores, semaphore)
	}

	for _, nc0 := range nodesCases {
		for _, nc1 := range nodesCases {
			if nc0.n == nc1.n {
				continue
			}
			nc0.n.connect(nc1.n)
		}
	}

	for _, n := range nodesCases {
		nd := n
		for _, v := range nd.values {
			nd.n.receive(message{v: v, knowledgeable: make(map[*node]struct{})})
		}
		if print {
			fmt.Printf("gossip of node [ %v ] done \n", nd)
		}
	}

	if print {
		fmt.Println("")
		fmt.Println("gossip in done")
		fmt.Println("")
	}

	// waits until all semaphores have closed channels.
	for _, s := range semaphores {
		for range s {
		}
	}

	cancel()

	for _, nc := range nodesCases {
		if print {
			fmt.Printf("\nValidating node [ %p ]\n", nc.n)
		}
		accumulate := make([]int, 0, upperValuesRange)
		for i := 0; i < upperValuesRange+1; i++ {
			if _, ok := nc.n.messages[i]; !ok {
				if print {
					fmt.Printf("    - [ %v ] not found\n", i)
				}
				continue
			}
			accumulate = append(accumulate, i)
		}
		if print {
			fmt.Printf("Node [ %p ] accumulated [ %v ] messages.\n", nc.n, len(accumulate))
		}
	}
}
