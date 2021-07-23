package runners

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/topolvm/pvc-autoresizer/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("test clientWrapper", func() {
	It("test metrics", func() {
		m := *mgr
		wrappedClient := client.NewClientWrapper(m.GetClient())
		pvc := corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "clientWrapperTest",
				Namespace: "default",
			},
		}
		err := wrappedClient.Update(context.TODO(), &pvc)
		Expect(err).To(HaveOccurred())
		mfs, err := getMetricsFamily()
		Expect(err).NotTo(HaveOccurred())
		mf, ok := mfs["pvcautoresizer_kubernetes_client_fail_total"]
		Expect(ok).To(BeTrue())

		var value int
		for _, m := range mf.Metric {
			labels := map[string]string{
				"group":   "",
				"version": "v1",
				"kind":    "PersistentVolumeClaim",
				"verb":    "PUT",
			}
			if !haveLabels(m, labels) {
				continue
			}
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

func getMetricsFamily() (map[string]*dto.MetricFamily, error) {
	resp, err := http.Get("http://localhost:8080/metrics")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parser expfmt.TextParser
	return parser.TextToMetricFamilies(resp.Body)
}

func haveLabels(m *dto.Metric, labels map[string]string) bool {
OUTER:
	for k, v := range labels {
		for _, label := range m.Label {
			if k == *label.Name && v == *label.Value {
				continue OUTER
			}
		}
		return false
	}
	return true
}
