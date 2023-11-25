package providers

import (
	"time"

	"github.com/bartossh/Computantis/src/transaction"
)

// HistogramProvider provides histogram telemetry capabilietes.
type HistogramProvider interface {
	CreateUpdateObservableHistogtram(name, description string)
	RecordHistogramTime(name string, t time.Duration) bool
	RecordHistogramValue(name string, f float64) bool
}

// GaugeProvider provides gauge telemetry capabilites.
type GaugeProvider interface {
	CreateUpdateObservableGauge(name, description string)
	AddToGauge(name string, f float64) bool
	RemoveFromGauge(name string, f float64) bool
	IncrementGauge(name string) bool
	DecrementGauge(name string) bool
	SetGauge(name string, f float64) bool
	SetToCurrentTimeGauge(name string) bool
}

// AwaitedTrxCacheProvider provides the cache functionality
type AwaitedTrxCacheProvider interface {
	SaveAwaitedTransaction(trx *transaction.Transaction) error
	RemoveAwaitedTransaction(hash [32]byte, address string) (transaction.Transaction, error)
	ReadTransactions(address string) ([]transaction.Transaction, error)
}
