package synchronizerserver

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/protobufcompiled"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ping struct {
	mux       sync.Mutex
	isProcess bool
}

func (p *ping) isProcessing() bool {
	p.mux.Lock()
	defer p.mux.Unlock()
	return p.isProcess
}

func (p *ping) setProcessing() {
	p.mux.Lock()
	defer p.mux.Unlock()
	p.isProcess = true
}

func (p *ping) unsetProcessing() {
	p.mux.Lock()
	defer p.mux.Unlock()
	p.isProcess = false
}

type node struct {
	id     string
	load   uint64
	conn   *grpc.ClientConn
	client protobufcompiled.QueueListenerClient
}

type queue struct {
	chAdd      chan node
	chRemove   chan string
	chPing     chan protobufcompiled.QueueListenerClient
	log        logger.Logger
	nodesQueue []node
}

func newQueue(log logger.Logger) *queue {
	return &queue{
		chAdd:      make(chan node, 1000),
		chRemove:   make(chan string, 1000),
		chPing:     make(chan protobufcompiled.QueueListenerClient, 1000),
		log:        log,
		nodesQueue: make([]node, 0, 1000),
	}
}

func (q *queue) addNode(info *protobufcompiled.NodeInfo) error {
	conn, err := grpc.Dial(info.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	client := protobufcompiled.NewQueueListenerClient(conn)
	q.chAdd <- node{
		id:     info.Id,
		load:   info.TransactionsPerSecond,
		conn:   conn,
		client: client,
	}
	return nil
}

func (q *queue) removeNode(info *protobufcompiled.NodeInfo) {
	q.chRemove <- info.Id
}

func (q *queue) processAddNode(n node) {
	q.nodesQueue = append(q.nodesQueue, n)
	slices.SortStableFunc(q.nodesQueue[1:], func(a, b node) int {
		return cmp.Compare(b.load, a.load) // decreasing order
	})
}

func (q *queue) processRemoveNode(id string) (node, bool) {
	q.nodesQueue = slices.DeleteFunc(q.nodesQueue, func(inner node) bool {
		if inner.id == id {
			go inner.conn.Close()
			return true
		}
		return false
	})
	if len(q.nodesQueue) == 0 {
		return node{}, false
	}
	return q.nodesQueue[0], true
}

func (q *queue) processInformFirstNode(ctx context.Context, n node) {
	_, err := n.client.QueueUpdate(ctx, &protobufcompiled.QueueStatus{
		IdOfFirstNodeInQueue:      n.id,
		TotalNumberOfNodesInQueue: uint64(len(q.nodesQueue)),
	})
	if err != nil {
		q.chRemove <- n.id
		q.log.Error(fmt.Sprintf("synchronizer queue cannot update the first node [ %s ] error: %s", n.id, err))
	}
}

func (q *queue) processPing(ctx context.Context, p *ping, nodes []node) {
	if p.isProcessing() {
		return
	}
	p.setProcessing()
	defer p.unsetProcessing()
	for _, n := range nodes {
		select {
		case <-ctx.Done():
			return
		default:
			if _, err := n.client.Ping(ctx, &emptypb.Empty{}); err != nil {
				q.log.Error(fmt.Sprintf("queue ping node [ %s ] error: %s", n.id, err))
				q.chRemove <- n.id
			}
		}
	}
}

func (q *queue) run(ctx context.Context, tick time.Duration) {
	var p ping
	t := time.NewTicker(tick)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			go q.processPing(ctx, &p, slices.Clone(q.nodesQueue))
		case n := <-q.chAdd:
			q.processAddNode(n)
		case id := <-q.chRemove:
			if first, ok := q.processRemoveNode(id); ok {
				q.processInformFirstNode(ctx, first)
			}
		}
	}
}
