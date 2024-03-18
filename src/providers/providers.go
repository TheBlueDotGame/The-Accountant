package providers

import (
	"time"

	"github.com/bartossh/Computantis/src/spice"
	"github.com/bartossh/Computantis/src/transaction"
)

// HistogramProvider provides histogram telemetry capabilietes.
type HistogramProvider interface {
	CreateUpdateObservableHistogram(name, description string)
	RecordHistogramTime(name string, t time.Duration) bool
	RecordHistogramValue(name string, f float64) bool
}

// GaugeProvider provides gauge telemetry capabilities.
type GaugeProvider interface {
	CreateUpdateObservableGauge(name, description string)
	AddToGauge(name string, f float64) bool
	RemoveFromGauge(name string, f float64) bool
	IncrementGauge(name string) bool
	DecrementGauge(name string) bool
	SetGauge(name string, f float64) bool
	SetToCurrentTimeGauge(name string) bool
}

// AwaitedTrxCacheProvider provides the cache functionality.
type AwaitedTrxCacheProvider interface {
	SaveAwaitedTransaction(trx *transaction.Transaction) error
	RemoveAwaitedTransaction(hash [32]byte, address string) (transaction.Transaction, error)
	ReadTransactions(address string) ([]transaction.Transaction, error)
}

// BalanceCacher is a balance cache provider.
type BalanceCacher interface {
	SaveBalance(a string, s spice.Melange) error
	ReadBalance(a string) (spice.Melange, error)
	RemoveBalance(a string) error
}

// AwaitedTrxCacheProviderBalanceCacher compounds the cache functionality for transaction and balance.
type AwaitedTrxCacheProviderBalanceCacher interface {
	AwaitedTrxCacheProvider
	BalanceCacher
}

// FlashbackMemoryProvider provides very short flashback memory about the hash.
type FlashbackMemoryHashProvider interface {
	HasHash(h []byte) (bool, error)
}

// FlashbackBalanceAddressProvider provides the address flashback checker.
type FlashbackBalanceAddressProvider interface {
	HasAddress(h string) (bool, error)
}

// FlashbackMemoryAddressRemover provides the address flashback remover.
type FlashbackMemoryAddressRemover interface {
	RemoveAddress(a string) error
}

// FlashbackMemoryHashProviderAddressRemover compounds memory hash checker and address remover.
type FlashbackMemoryHashProviderAddressRemover interface {
	FlashbackMemoryHashProvider
	FlashbackMemoryAddressRemover
}

// FlashbackMemoryAddressProvideRemover compounds memory hash and address checker.
type FlashbackMemoryAddressProvideRemover interface {
	FlashbackBalanceAddressProvider
	FlashbackMemoryAddressRemover
}
