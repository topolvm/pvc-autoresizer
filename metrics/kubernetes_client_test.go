package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestKubernetesClientFailTotal(t *testing.T) {
	KubernetesClientFailTotal.Increment()
	actual := testutil.ToFloat64(kubernetesClientFailTotal)
	if actual != float64(1) {
		t.Fatalf("value is not %d", 1)
	}

	KubernetesClientFailTotal.Increment()
	actual = testutil.ToFloat64(kubernetesClientFailTotal)
	if actual != float64(2) {
		t.Fatalf("value is not %d", 2)
	}

	actual2 := testutil.CollectAndCount(kubernetesClientFailTotal)
	if actual2 != 1 {
		t.Fatalf("the count of metrics is not 1 actual=%d", actual2)
	}
}
