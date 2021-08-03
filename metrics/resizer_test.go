package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestResizerSuccessLoopTotal(t *testing.T) {
	ns := "default"
	name := "pvc"

	ResizerSuccessLoopTotal.Increment(ns, name)
	actual := testutil.ToFloat64(resizerSuccessLoopTotal.WithLabelValues(ns, name))
	if actual != float64(1) {
		t.Fatalf("namespace=%s name=%s value is not %d", ns, name, 1)
	}
}

func TestResizerFailedLoopTotal(t *testing.T) {
	ns := "default"
	name := "pvc"

	ResizerFailedLoopTotal.Increment(ns, name)
	actual := testutil.ToFloat64(resizerFailedLoopTotal.WithLabelValues(ns, name))
	if actual != float64(1) {
		t.Fatalf("namespace=%s name=%s value is not %d", ns, name, 1)
	}
}

func TestResizerLoopSecondsTotal(t *testing.T) {
	ns := "default"
	name := "pvc"

	ResizerLoopSecondsTotal.Add(10)
	actual := testutil.ToFloat64(resizerLoopSecondsTotal.(prometheus.Collector))
	if actual != float64(10) {
		t.Fatalf("namespace=%s name=%s value is not %d", ns, name, 10)
	}
}
