package runners

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
)

type prometheusClientMock struct {
	stats map[types.NamespacedName]*VolumeStats
	mutex sync.Mutex
}

func (c *prometheusClientMock) GetMetrics(ctx context.Context) (map[types.NamespacedName]*VolumeStats, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	copied := make(map[types.NamespacedName]*VolumeStats)
	for k, v := range c.stats {
		copied[k] = v
	}
	return copied, nil
}

func (c *prometheusClientMock) setResponce(key types.NamespacedName, stats *VolumeStats) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.stats == nil {
		c.stats = make(map[types.NamespacedName]*VolumeStats)
	}
	c.stats[key] = stats
}

var _ = Describe("test prometheusClient", func() {
	It("test metrics", func() {
		ts := httptest.NewServer(http.HandlerFunc(http.NotFound))
		defer ts.Close()

		c, err := NewPrometheusClient(ts.URL, "", logr.Discard())
		Expect(err).ToNot(HaveOccurred())
		_, err = c.GetMetrics(context.TODO())
		Expect(err).To(HaveOccurred())

		mfs, err := getMetricsFamily()
		Expect(err).NotTo(HaveOccurred())
		mf, ok := mfs["pvcautoresizer_metrics_client_fail_total"]
		Expect(ok).To(BeTrue())

		var value int
		for _, m := range mf.Metric {
			if m.Counter == nil {
				continue
			}
			if m.Counter.Value == nil {
				continue
			}
			value = int(*m.Counter.Value)
		}
		Expect(value).NotTo(Equal(0))
	})
})
