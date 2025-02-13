package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	runtimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Metrics subsystem and all of the keys used by the resizer.
const (
	ResizerSuccessResizeTotalKey           = "success_resize_total"
	ResizerFailedResizeTotalKey            = "failed_resize_total"
	ResizerLoopSecondsTotalKey             = "loop_seconds_total"
	ResizerLimitReachedTotalKey            = "limit_reached_total"
	ResizerSuccessPatchAnnotationsTotalKey = "success_patch_annotations_total"
	ResizerFailedPatchAnnotationsTotalKey  = "failed_patch_annotations_total"
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

type resizerSuccessPatchAnnotationsTotalAdapter struct {
	metric prometheus.CounterVec
}

func (a *resizerSuccessPatchAnnotationsTotalAdapter) Increment(pvcName string, pvcNamespace string) {
	a.metric.With(prometheus.Labels{"persistentvolumeclaim": pvcName, "namespace": pvcNamespace}).Inc()
}

// SpecifyLabels helps output metrics before the first limit reached event of resize.
// This method specifies the metric labels and add 0 to the metric value.
func (a *resizerSuccessPatchAnnotationsTotalAdapter) SpecifyLabels(pvcName string, pvcNamespace string) {
	a.metric.With(prometheus.Labels{"persistentvolumeclaim": pvcName, "namespace": pvcNamespace}).Add(0)
}

type resizerFailedPatchAnnotationsTotalAdapter struct {
	metric prometheus.CounterVec
}

func (a *resizerFailedPatchAnnotationsTotalAdapter) Increment(pvcName string, pvcNamespace string) {
	a.metric.With(prometheus.Labels{"persistentvolumeclaim": pvcName, "namespace": pvcNamespace}).Inc()
}

// SpecifyLabels helps output metrics before the first limit reached event of resize.
// This method specifies the metric labels and add 0 to the metric value.
func (a *resizerFailedPatchAnnotationsTotalAdapter) SpecifyLabels(pvcName string, pvcNamespace string) {
	a.metric.With(prometheus.Labels{"persistentvolumeclaim": pvcName, "namespace": pvcNamespace}).Add(0)
}

var (
	resizerSuccessResizeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      ResizerSuccessResizeTotalKey,
		Help:      "counter that indicates how many volume expansion processing resizes succeed.",
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

	resizerSuccessPatchAnnotationsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      ResizerSuccessPatchAnnotationsTotalKey,
		Help:      "counter that indicates how many annotation patches on StatefulSet provisioned PersistentVolumeClaims succeed.",
	}, []string{"persistentvolumeclaim", "namespace"})

	resizerFailedPatchAnnotationsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      ResizerFailedPatchAnnotationsTotalKey,
		Help:      "counter that indicates how many annotation patches on StatefulSet provisioned PersistentVolumeClaims fail.",
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
	ResizerSuccessPatchAnnotationsTotal *resizerSuccessPatchAnnotationsTotalAdapter = &resizerSuccessPatchAnnotationsTotalAdapter{
		metric: *resizerSuccessPatchAnnotationsTotal,
	}
	ResizerFailedPatchAnnotationsTotal *resizerFailedPatchAnnotationsTotalAdapter = &resizerFailedPatchAnnotationsTotalAdapter{
		metric: *resizerFailedPatchAnnotationsTotal,
	}
)

func registerResizerMetrics() {
	runtimemetrics.Registry.MustRegister(resizerSuccessResizeTotal)
	runtimemetrics.Registry.MustRegister(resizerFailedResizeTotal)
	runtimemetrics.Registry.MustRegister(resizerLoopSecondsTotal)
	runtimemetrics.Registry.MustRegister(resizerLimitReachedTotal)
	runtimemetrics.Registry.MustRegister(resizerSuccessPatchAnnotationsTotal)
	runtimemetrics.Registry.MustRegister(resizerFailedPatchAnnotationsTotal)
}
