package metrics

import (
	"bytes"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	runtimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Metrics subsystem and all of the keys used by the resizer.
const (
	ResizerSuccessResizeTotalKey = "success_resize_total"
	ResizerFailedResizeTotalKey  = "failed_resize_total"
	ResizerLoopSecondsTotalKey   = "loop_seconds_total"
	ResizerLimitReachedTotalKey  = "limit_reached_total"
)

func init() {
	registerResizerMetrics()
}

type resizerSuccessResizeTotalAdapter struct {
	metric prometheus.CounterVec
}

func (a *resizerSuccessResizeTotalAdapter) Increment(pvcname string, pvcns string) {
	a.metric.With(prometheus.Labels{"persistentvolumeclaim": pvcname, "namespace": pvcns}).Inc()
}

// SpecifyLabels helps output metrics before the first resize event.
// This method specifies the metric labels and add 0 to the metric value.
func (a *resizerSuccessResizeTotalAdapter) SpecifyLabels(pvcname string, pvcns string) {
	a.metric.With(prometheus.Labels{"persistentvolumeclaim": pvcname, "namespace": pvcns}).Add(0)
}

type resizerFailedResizeTotalAdapter struct {
	metric prometheus.CounterVec
}

func (a *resizerFailedResizeTotalAdapter) Increment(pvcname string, pvcns string) {
	a.metric.With(prometheus.Labels{"persistentvolumeclaim": pvcname, "namespace": pvcns}).Inc()
}

// SpecifyLabels helps output metrics before the first fail event of resize.
// This method specifies the metric labels and add 0 to the metric value.
func (a *resizerFailedResizeTotalAdapter) SpecifyLabels(pvcname string, pvcns string) {
	a.metric.With(prometheus.Labels{"persistentvolumeclaim": pvcname, "namespace": pvcns}).Add(0)
}

type resizerLoopSecondsTotalAdapter struct {
	metric prometheus.Counter
}

func (a *resizerLoopSecondsTotalAdapter) Add(value float64) {
	a.metric.Add(value)
}

type resizerLimitReachedTotalAdapter struct {
	metric prometheus.CounterVec
}

func (a *resizerLimitReachedTotalAdapter) Increment(pvcname string, pvcns string) {
	a.metric.With(prometheus.Labels{"persistentvolumeclaim": pvcname, "namespace": pvcns}).Inc()
}

// SpecifyLabels helps output metrics before the first limit reached event of resize.
// This method specifies the metric labels and add 0 to the metric value.
func (a *resizerLimitReachedTotalAdapter) SpecifyLabels(pvcname string, pvcns string) {
	a.metric.With(prometheus.Labels{"persistentvolumeclaim": pvcname, "namespace": pvcns}).Add(0)
}

var (
	resizerSuccessResizeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      ResizerSuccessResizeTotalKey,
		Help:      "counter that indicates how many volume expansion processing resized succeed.",
	}, []string{"persistentvolumeclaim", "namespace"})

	resizerFailedResizeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      ResizerFailedResizeTotalKey,
		Help:      "counter that indicates how many volume expansion processing resizes fail.",
	}, []string{"persistentvolumeclaim", "namespace"})

	resizerLoopSecondsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      ResizerLoopSecondsTotalKey,
		Help:      "counter that indicates the sum of seconds spent on volume expansion processing loops.",
	})

	resizerLimitReachedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      ResizerLimitReachedTotalKey,
		Help:      "counter that indicates how many storage limits were reached.",
	}, []string{"persistentvolumeclaim", "namespace"})

	ResizerSuccessResizeTotal *resizerSuccessResizeTotalAdapter = &resizerSuccessResizeTotalAdapter{
		metric: *resizerSuccessResizeTotal,
	}
	ResizerFailedResizeTotal *resizerFailedResizeTotalAdapter = &resizerFailedResizeTotalAdapter{
		metric: *resizerFailedResizeTotal,
	}
	ResizerLoopSecondsTotal *resizerLoopSecondsTotalAdapter = &resizerLoopSecondsTotalAdapter{
		metric: resizerLoopSecondsTotal,
	}
	ResizerLimitReachedTotal *resizerLimitReachedTotalAdapter = &resizerLimitReachedTotalAdapter{
		metric: *resizerLimitReachedTotal,
	}
)

func registerResizerMetrics() {
	runtimemetrics.Registry.MustRegister(resizerSuccessResizeTotal)
	runtimemetrics.Registry.MustRegister(resizerFailedResizeTotal)
	runtimemetrics.Registry.MustRegister(resizerLoopSecondsTotal)
	runtimemetrics.Registry.MustRegister(resizerLimitReachedTotal)
}

// currentMetricsSizeBytes returns the byte size of all metrics encoded in the
// Prometheus text format, matching the payload used to decide when to reset.
func currentMetricsSizeBytes() (uint64, error) {
	mfs, err := runtimemetrics.Registry.Gather()
	if err != nil {
		return 0, err
	}

	var buf bytes.Buffer
	enc := expfmt.NewEncoder(&buf, expfmt.NewFormat(expfmt.TypeTextPlain))
	for _, mf := range mfs {
		if err := enc.Encode(mf); err != nil {
			return 0, err
		}
	}
	return uint64(buf.Len()), nil
}

func resetLabelHeavyMetrics() {
	resizerSuccessResizeTotal.Reset()
	resizerFailedResizeTotal.Reset()
	resizerLimitReachedTotal.Reset()
}

// ResetMetricsIfExceedsThreshold checks the total size of all registered metrics and
// resets label-heavy resizer metrics if their encoded size exceeds thresholdBytes.
// If thresholdBytes is 0, reset is disabled and this returns false, nil.
func ResetMetricsIfExceedsThreshold(thresholdBytes uint64) (bool, error) {
	if thresholdBytes == 0 {
		return false, nil
	}

	size, err := currentMetricsSizeBytes()
	if err != nil {
		return false, fmt.Errorf("failed to calculate metrics size: %w", err)
	}
	if size > thresholdBytes {
		resetLabelHeavyMetrics()
		return true, nil
	}
	return false, nil
}
