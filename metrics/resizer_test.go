package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestResizerSuccessResizeTotal(t *testing.T) {
	ResizerSuccessResizeTotal.Increment()
	actual := testutil.ToFloat64(resizerSuccessResizeTotal)
	if actual != float64(1) {
		t.Fatalf("value is not %d", 1)
	}
}

func TestResizerFailedResizeTotal(t *testing.T) {
	ResizerFailedResizeTotal.Increment()
	actual := testutil.ToFloat64(resizerFailedResizeTotal)
	if actual != float64(1) {
		t.Fatalf("value is not %d", 1)
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

func TestResizerLimitReachedTotal(t *testing.T) {
	ResizerLimitReachedTotal.Increment()
	actual := testutil.ToFloat64(resizerLimitReachedTotal)
	if actual != float64(1) {
		t.Fatalf("value is not %d", 1)
	}
}
