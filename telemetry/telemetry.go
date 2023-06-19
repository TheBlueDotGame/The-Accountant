package telemetry

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "computantis_processed_ops_total",
		Help: "The total number of processed events",
	})
)

func recordMetrics() {
	for {
		opsProcessed.Inc()
		time.Sleep(2 * time.Second)
	}
}

// Run starts collecting metrics and server with prometheus telemetry endpoint.
// This functions blocks. To stop cancel ctx.
func Run(ctx context.Context, cancel context.CancelFunc) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := http.Server{Addr: ":2112", Handler: mux}

	var err error
	go func() {
		if err = srv.ListenAndServe(); err != nil {
			cancel()
			return
		}
	}()

	go recordMetrics()

	<-ctx.Done()

	err = srv.Shutdown(ctx)
	return err
}
