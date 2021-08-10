package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	runtimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Metrics subsystem and all of the keys used by the resizer.
const (
	ResizerSuccessResizeTotalKey = "success_resize_total"
	ResizerFailedResizeTotalKey  = "failed_resize_total"
	ResizerLoopSecondsTotalKey   = "loop_seconds_total"
)

func init() {
	registerResizerMetrics()
}

type resizerSuccessResizeTotalAdapter struct {
	metric prometheus.Counter
}

func (a *resizerSuccessResizeTotalAdapter) Increment() {
	a.metric.Inc()
}

type resizerFailedResizeTotalAdapter struct {
	metric prometheus.Counter
}

func (a *resizerFailedResizeTotalAdapter) Increment() {
	a.metric.Inc()
}

type resizerLoopSecondsTotalAdapter struct {
	metric prometheus.Counter
}

func (a *resizerLoopSecondsTotalAdapter) Add(value float64) {
	a.metric.Add(value)
}

var (
	resizerSuccessResizeTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      ResizerSuccessResizeTotalKey,
		Help:      "counter that indicates how many volume expansion processing resizes succeed.",
	})

	resizerFailedResizeTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      ResizerFailedResizeTotalKey,
		Help:      "counter that indicates how many volume expansion processing resizes fail.",
	})

	resizerLoopSecondsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      ResizerLoopSecondsTotalKey,
		Help:      "counter that indicates the sum of seconds spent on volume expansion processing loops.",
	})

	ResizerSuccessResizeTotal *resizerSuccessResizeTotalAdapter = &resizerSuccessResizeTotalAdapter{metric: resizerSuccessResizeTotal}
	ResizerFailedResizeTotal  *resizerFailedResizeTotalAdapter  = &resizerFailedResizeTotalAdapter{metric: resizerFailedResizeTotal}
	ResizerLoopSecondsTotal   *resizerLoopSecondsTotalAdapter   = &resizerLoopSecondsTotalAdapter{metric: resizerLoopSecondsTotal}
)

func registerResizerMetrics() {
	runtimemetrics.Registry.MustRegister(resizerSuccessResizeTotal)
	runtimemetrics.Registry.MustRegister(resizerFailedResizeTotal)
	runtimemetrics.Registry.MustRegister(resizerLoopSecondsTotal)
}
