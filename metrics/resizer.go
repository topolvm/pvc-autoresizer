package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	runtimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Metrics subsystem and all of the keys used by the resizer.
const (
	ResizerSubsystem           = "resizer"
	ResizerSuccessLoopTotalKey = "success_loop_total"
	ResizerFailedLoopTotalKey  = "failed_loop_total"
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

var (
	resizerSuccessLoopTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: ResizerSubsystem,
		Name:      ResizerSuccessLoopTotalKey,
		Help:      "counter that indicates how many volume expansion processing loops succeed.",
	}, []string{"namespace", "name"})

	resizerFailedLoopTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: ResizerSubsystem,
		Name:      ResizerFailedLoopTotalKey,
		Help:      "counter that indicates how many volume expansion processing loops are failed.",
	}, []string{"namespace", "name"})

	ResizerSuccessLoopTotal *resizerSuccessLoopTotalAdapter = &resizerSuccessLoopTotalAdapter{metric: resizerSuccessLoopTotal}
	ResizerFailesLoopTotal  *resizerFailedLoopTotalAdapter  = &resizerFailedLoopTotalAdapter{metric: resizerFailedLoopTotal}
)

func registerResizerMetrics() {
	runtimemetrics.Registry.MustRegister(resizerSuccessLoopTotal)
	runtimemetrics.Registry.MustRegister(resizerFailedLoopTotal)
}
