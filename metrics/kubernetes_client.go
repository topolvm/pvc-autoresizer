package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	runtimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Metrics subsystem and all of the keys used by the metrics client.
const (
	KubernetesClientSubsystem    = "kubernetes_client"
	KubernetesClientFailTotalKey = "fail_total"
)

func init() {
	registerKubernetesClientMetrics()
}

type KubernetesClientFailTotalAdapter struct {
	metric *prometheus.CounterVec
}

func (a *KubernetesClientFailTotalAdapter) Increment(group, version, kind, verb string) {
	a.metric.WithLabelValues(group, version, kind, verb).Inc()
}

var (
	kubernetesClientFailTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Subsystem: KubernetesClientSubsystem,
		Name:      KubernetesClientFailTotalKey,
		Help:      "counter that indicates how many API requests to kube-api server are failed.",
	}, []string{"group", "version", "kind", "verb"})

	KubernetesClientFailTotal *KubernetesClientFailTotalAdapter = &KubernetesClientFailTotalAdapter{metric: kubernetesClientFailTotal}
)

func registerKubernetesClientMetrics() {
	runtimemetrics.Registry.MustRegister(kubernetesClientFailTotal)
}
