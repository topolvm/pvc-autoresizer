package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	runtimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Metrics subsystem and all of the keys used by the metrics client.
const (
	MetricsClientSubsystem    = "metrics_client"
	MetricsClientFailTotalKey = "fail_total"
)

func init() {
	registerMetricsClientMetrics()
}

type metricsClientFailTotalAdapter struct {
	metric *prometheus.CounterVec
}

func (a *metricsClientFailTotalAdapter) Increment(address, query string) {
	a.metric.WithLabelValues(address, query).Inc()
}

var (
	metricsClientFailTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: MetricsClientSubsystem,
		Name:      MetricsClientFailTotalKey,
		Help:      "counter that indicates how many API requests to metrics server(e.g. prometheus) are failed.",
	}, []string{"address", "query"})

	FailTotal *metricsClientFailTotalAdapter = &metricsClientFailTotalAdapter{metric: metricsClientFailTotal}
)

func registerMetricsClientMetrics() {
	runtimemetrics.Registry.MustRegister(metricsClientFailTotal)
}
