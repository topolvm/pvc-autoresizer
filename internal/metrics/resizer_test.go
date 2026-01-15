package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestResizerSuccessResizeTotal(t *testing.T) {
	ResizerSuccessResizeTotal.Increment("my-test-pvc", "my-test-namespace")
	actual := testutil.ToFloat64(resizerSuccessResizeTotal)
	if actual != float64(1) {
		t.Fatalf("value is not %d", 1)
	}
}

func TestResetMetricsIfExceedsThreshold(t *testing.T) {
	ResizerSuccessResizeTotal.Increment("pvc-a", "ns-a")
	ResizerFailedResizeTotal.Increment("pvc-a", "ns-a")
	ResizerLimitReachedTotal.Increment("pvc-a", "ns-a")

	before := testutil.CollectAndCount(resizerSuccessResizeTotal)
	if before == 0 {
		t.Fatalf("expected metrics before reset")
	}

	size, err := currentMetricsSizeBytes()
	if err != nil {
		t.Fatalf("failed to get metrics size: %v", err)
	}

	if size == 0 {
		t.Fatalf("expected metrics size to be > 0")
	}

	reset, err := ResetMetricsIfExceedsThreshold(size - 1)
	if err != nil {
		t.Fatalf("ResetMetricsIfExceedsThreshold returned error: %v", err)
	}
	if !reset {
		t.Fatalf("expected reset to occur")
	}

	after := testutil.CollectAndCount(resizerSuccessResizeTotal)
	if after != 0 {
		t.Fatalf("expected metrics to be cleared, got %d", after)
	}
}

func TestResetMetricsIfExceedsThreshold_NoResetBelowThreshold(t *testing.T) {
	resetLabelHeavyMetrics()
	defer resetLabelHeavyMetrics()

	ResizerSuccessResizeTotal.Increment("pvc-a", "ns-a")

	before := testutil.CollectAndCount(resizerSuccessResizeTotal)
	if before == 0 {
		t.Fatalf("expected metrics before reset")
	}

	size, err := currentMetricsSizeBytes()
	if err != nil {
		t.Fatalf("failed to get metrics size: %v", err)
	}

	reset, err := ResetMetricsIfExceedsThreshold(size + 100)
	if err != nil {
		t.Fatalf("ResetMetricsIfExceedsThreshold returned error: %v", err)
	}
	if reset {
		t.Fatalf("expected reset to not occur")
	}

	after := testutil.CollectAndCount(resizerSuccessResizeTotal)
	if after != before {
		t.Fatalf("expected metrics to remain, got %d (want %d)", after, before)
	}
}

func TestResizerFailedResizeTotal(t *testing.T) {
	ResizerFailedResizeTotal.Increment("my-test-pvc", "my-test-namespace")
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
	ResizerLimitReachedTotal.Increment("my-test-pvc", "my-test-namespace")
	actual := testutil.ToFloat64(resizerLimitReachedTotal)
	if actual != float64(1) {
		t.Fatalf("value is not %d", 1)
	}
}
