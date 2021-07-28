package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestKubernetesClientFailTotal(t *testing.T) {
	group := ""
	version := "v1"
	kind := "Pod"
	verb := "LIST"

	KubernetesClientFailTotal.Increment(group, version, kind, verb)
	actual := testutil.ToFloat64(kubernetesClientFailTotal.WithLabelValues(group, version, kind, verb))
	if actual != float64(1) {
		t.Fatalf("group=%s version=%s kind=%s verb=%s value is not %d", group, version, kind, verb, 1)
	}

	KubernetesClientFailTotal.Increment(group, version, kind, verb)
	actual = testutil.ToFloat64(kubernetesClientFailTotal.WithLabelValues("", "v1", "Pod", "LIST"))
	if actual != float64(2) {
		t.Fatalf("group=%s version=%s kind=%s verb=%s value is not %d", group, version, kind, verb, 2)
	}

	actual2 := testutil.CollectAndCount(kubernetesClientFailTotal)
	if actual2 != 1 {
		t.Fatalf("the count of metrics is not 1 actual=%d", actual2)
	}
}
