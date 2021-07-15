package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsClientFailTotal(t *testing.T) {
	address := "http://localhost"
	query := "test_query"
	query2 := "test_query2"

	MetricsClientFailTotal.SetAddress(address)
	MetricsClientFailTotal.Increment(query)
	actual := testutil.ToFloat64(metricsClientFailTotal.WithLabelValues(address, query))
	if actual != float64(1) {
		t.Fatalf("address=%s query=%s value is not %d", address, query, 1)
	}

	MetricsClientFailTotal.Increment(query)
	actual = testutil.ToFloat64(metricsClientFailTotal.WithLabelValues(address, query))
	if actual != float64(2) {
		t.Fatalf("address=%s query=%s value is not %d", address, query, 2)
	}

	MetricsClientFailTotal.Increment(query2)
	actual = testutil.ToFloat64(metricsClientFailTotal.WithLabelValues(address, query2))
	if actual != float64(1) {
		t.Fatalf("address=%s query=%s value is not %d", address, query2, 1)
	}

	actual2 := testutil.CollectAndCount(metricsClientFailTotal)
	if actual2 != 2 {
		t.Fatalf("the count of metrics is not 1 actual=%d", actual2)
	}
}
