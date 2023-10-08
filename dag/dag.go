package dag

import "github.com/bartossh/Computantis/transaction"

// Vertax is an entity in the Graph that is approved by the Tip.
type Vertex struct {
	Transaction   transaction.Transaction
	Hash          [32]byte
	PrevHashLeft  [32]byte
	PrevHashRight [32]byte
}

// TransactionGraph is the cryptographically secured transaction directed acyclic graph.
type TransactionGraph struct{}
