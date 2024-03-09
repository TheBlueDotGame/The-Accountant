package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type record struct {
	name  byte
	value float64
}

// Measurements collects measurements for prometheus.
type Measurements struct {
	histograms map[string]prometheus.Observer
	gauge      map[string]prometheus.Gauge
}

// CreateUpdateObservableHistogtram creats or updates observable histogram.
func (m *Measurements) CreateUpdateObservableHistogtram(name, description string) {
	hist := promauto.NewHistogram(prometheus.HistogramOpts{
		Name: name,
		Help: description,
	})

	m.histograms[name] = hist
}

// RecordHistogramTime records histogram time if entity with given name exists.
func (m *Measurements) RecordHistogramTime(name string, t time.Duration) bool {
	ts := float64(t.Microseconds())
	if v, ok := m.histograms[name]; ok {
		v.Observe(ts)
		return true
	}
	return false
}

// RecordHistogramValue records histogram value if entity with given name exists.
func (m *Measurements) RecordHistogramValue(name string, f float64) bool {
	if v, ok := m.histograms[name]; ok {
		v.Observe(f)
		return true
	}
	return false
}

// CreateUpdateObservableGauge creats or updates observable gauge.
func (m *Measurements) CreateUpdateObservableGauge(name, description string) {
	gauge := promauto.NewGauge(prometheus.GaugeOpts{
		Name: name,
		Help: description,
	})

	m.gauge[name] = gauge
}

// AddToGeuge adds to gauge the value if entity with given name exists.
func (m *Measurements) AddToGauge(name string, f float64) bool {
	if v, ok := m.gauge[name]; ok {
		v.Add(f)
		return true
	}
	return false
}

// SubstractFromGeuge substracts from gauge the value if entity with given name exists.
func (m *Measurements) RemoveFromGauge(name string, f float64) bool {
	if v, ok := m.gauge[name]; ok {
		v.Sub(f)
		return true
	}
	return false
}

// IncrementGeuge increments gauge the value if entity with given name exists.
func (m *Measurements) IncrementGauge(name string) bool {
	if v, ok := m.gauge[name]; ok {
		v.Inc()
		return true
	}
	return false
}

// DecrementGeuge decrements gauge the value if entity with given name exists.
func (m *Measurements) DecrementGauge(name string) bool {
	if v, ok := m.gauge[name]; ok {
		v.Dec()
		return true
	}
	return false
}

// SetGeuge sets the gauge to the value if entity with given name exists.
func (m *Measurements) SetGauge(name string, f float64) bool {
	if v, ok := m.gauge[name]; ok {
		v.Set(f)
		return true
	}
	return false
}

// SetToCurrentTimeGeuge sets the gauge to the current time if entity with given name exists.
func (m *Measurements) SetToCurrentTimeGauge(name string) bool {
	if v, ok := m.gauge[name]; ok {
		v.SetToCurrentTime()
		return true
	}
	return false
}

// Run starts collecting metrics and server with prometheus telemetry endpoint.
// Returns Measurements structure if successfully started or cancels context otherwise.
// Default port of 2112 is used if port value is set to 0.
func Run(ctx context.Context, cancel context.CancelFunc, port int) (*Measurements, error) {
	if port > 65535 || port < 0 {
		return nil, fmt.Errorf("port range allowed is from 1 to 65535, received %d", port)
	}
	go func() {
		if port == 0 {
			port = 2112
		}
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		srv := http.Server{Addr: fmt.Sprintf(":%d", port), Handler: mux}

		var err error
		go func() {
			if err = srv.ListenAndServe(); err != nil {
				cancel()
			}
		}()

		<-ctx.Done()

		srv.Shutdown(ctx)
	}()

	return &Measurements{make(map[string]prometheus.Observer), make(map[string]prometheus.Gauge)}, nil
}
