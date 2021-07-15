package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	runtimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Metrics subsystem and all of the keys used by the resizer.
const (
	ResizerSuccessLoopTotalKey = "success_loop_total"
	ResizerFailedLoopTotalKey  = "failed_loop_total"
	ResizerLoopSecondsTotalKey = "loop_seconds_total"
)

func init() {
	registerResizerMetrics()
}

type resizerSuccessLoopTotalAdapter struct {
	metric *prometheus.CounterVec
}

func (a *resizerSuccessLoopTotalAdapter) Increment(ns, name string) {
	a.metric.WithLabelValues(ns, name).Inc()
}

type resizerFailedLoopTotalAdapter struct {
	metric *prometheus.CounterVec
}

func (a *resizerFailedLoopTotalAdapter) Increment(ns, name string) {
	a.metric.WithLabelValues(ns, name).Inc()
}

type loopSecondsTotalAdapter struct {
	metric prometheus.Counter
}

func (a *loopSecondsTotalAdapter) Add(value float64) {
	a.metric.Add(value)
}

var (
	resizerSuccessLoopTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      ResizerSuccessLoopTotalKey,
		Help:      "counter that indicates how many volume expansion processing loops succeed.",
	}, []string{"namespace", "name"})

	resizerFailedLoopTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      ResizerFailedLoopTotalKey,
		Help:      "counter that indicates how many volume expansion processing loops are failed.",
	}, []string{"namespace", "name"})

	resizerLoopSecondsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      ResizerLoopSecondsTotalKey,
		Help:      "counter that indicates the sum of seconds spent on volume expansion processing loops.",
	})

	ResizerSuccessLoopTotal *resizerSuccessLoopTotalAdapter = &resizerSuccessLoopTotalAdapter{metric: resizerSuccessLoopTotal}
	ResizerFailedLoopTotal  *resizerFailedLoopTotalAdapter  = &resizerFailedLoopTotalAdapter{metric: resizerFailedLoopTotal}
	ResizerLoopSecondsTotal *loopSecondsTotalAdapter        = &loopSecondsTotalAdapter{metric: resizerLoopSecondsTotal}
)

func registerResizerMetrics() {
	runtimemetrics.Registry.MustRegister(resizerSuccessLoopTotal)
	runtimemetrics.Registry.MustRegister(resizerFailedLoopTotal)
	runtimemetrics.Registry.MustRegister(resizerLoopSecondsTotal)
}
