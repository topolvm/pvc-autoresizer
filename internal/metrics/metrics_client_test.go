package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsClientFailTotal(t *testing.T) {
	MetricsClientFailTotal.Increment()
	actual := testutil.ToFloat64(metricsClientFailTotal.(prometheus.Collector))
	if actual != float64(1) {
		t.Fatalf("value is not %d", 1)
	}
}
