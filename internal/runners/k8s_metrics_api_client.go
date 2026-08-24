package runners

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/topolvm/pvc-autoresizer/internal/metrics"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// nodeMetricsRequestTimeout bounds a single node's kubelet-proxy request. Without it, a
// node that accepts the connection but never responds would block wg.Wait() forever,
// since removing errgroup's shared-cancellation also removed its only timeout behavior.
const nodeMetricsRequestTimeout = 10 * time.Second

// NewK8sMetricsApiClient returns a new k8sMetricsApiClient client
func NewK8sMetricsApiClient() (MetricsClient, error) {
	return &k8sMetricsApiClient{}, nil
}

type k8sMetricsApiClient struct {
}

func (c *k8sMetricsApiClient) GetMetrics(ctx context.Context) (map[types.NamespacedName]*VolumeStats, error) {
	// create a Kubernetes client using in-cluster configuration
	config, err := rest.InClusterConfig()
	if err != nil {
		metrics.MetricsClientFailTotal.Increment()
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		metrics.MetricsClientFailTotal.Increment()
		return nil, err
	}

	// get a list of nodes and IP addresses
	nodes, err := clientset.CoreV1().Nodes().List(ctx, v1.ListOptions{})
	if err != nil {
		metrics.MetricsClientFailTotal.Increment()
		return nil, err
	}

	// create a map to hold PVC usage data
	pvcUsage := make(map[types.NamespacedName]*VolumeStats)
	var mu sync.Mutex // serialize writes to pvcUsage

	// Query kubelet for PVC usage on each node independently. A node that fails to
	// respond (e.g. mid-scale-down) only loses its own PVC data; it must not abort
	// metrics collection for the rest of the cluster.
	var wg sync.WaitGroup
	for _, node := range nodes.Items {
		nodeName := node.Name
		wg.Add(1)
		go func() {
			defer wg.Done()
			nodeCtx, cancel := context.WithTimeout(ctx, nodeMetricsRequestTimeout)
			defer cancel()
			nodePVCUsage, err := getPVCUsageFromK8sMetricsAPI(nodeCtx, clientset, nodeName)
			if err != nil {
				metrics.MetricsClientFailTotal.Increment()
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for k, v := range nodePVCUsage {
				pvcUsage[k] = v
			}
		}()
	}
	wg.Wait()

	return pvcUsage, nil
}

func getPVCUsageFromK8sMetricsAPI(
	ctx context.Context, clientset *kubernetes.Clientset, nodeName string,
) (map[types.NamespacedName]*VolumeStats, error) {
	// make the request to the api /metrics endpoint and handle the response
	req := clientset.
		CoreV1().
		RESTClient().
		Get().
		Resource("nodes").
		Name(nodeName).
		SubResource("proxy").
		Suffix("metrics")
	respBody, err := req.DoRaw(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats from kubelet on node %s: %w", nodeName, err)
	}
	parser := expfmt.NewTextParser(model.UTF8Validation)
	metricFamilies, err := parser.TextToMetricFamilies(bytes.NewReader(respBody))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from kubelet on node %s: %w", nodeName, err)
	}

	pvcUsage := make(map[types.NamespacedName]*VolumeStats)

	// volumeAvailableQuery
	if gauge, ok := metricFamilies[volumeAvailableQuery]; ok {
		for _, m := range gauge.Metric {
			pvcName, value := parseMetric(m)
			pvcUsage[pvcName] = &VolumeStats{}
			pvcUsage[pvcName].AvailableBytes = int64(value)
		}
	}
	// volumeCapacityQuery
	if gauge, ok := metricFamilies[volumeCapacityQuery]; ok {
		for _, m := range gauge.Metric {
			pvcName, value := parseMetric(m)
			pvcUsage[pvcName].CapacityBytes = int64(value)
		}
	}

	// inodesAvailableQuery
	if gauge, ok := metricFamilies[inodesAvailableQuery]; ok {
		for _, m := range gauge.Metric {
			pvcName, value := parseMetric(m)
			pvcUsage[pvcName].AvailableInodeSize = int64(value)
		}
	}

	// inodesCapacityQuery
	if gauge, ok := metricFamilies[inodesCapacityQuery]; ok {
		for _, m := range gauge.Metric {
			pvcName, value := parseMetric(m)
			pvcUsage[pvcName].CapacityInodeSize = int64(value)
		}
	}
	return pvcUsage, nil
}

func parseMetric(m *dto.Metric) (pvcName types.NamespacedName, value uint64) {
	for _, label := range m.GetLabel() {
		if label.GetName() == "namespace" {
			pvcName.Namespace = label.GetValue()
		} else if label.GetName() == "persistentvolumeclaim" {
			pvcName.Name = label.GetValue()
		}
	}
	value = uint64(m.GetGauge().GetValue())
	return pvcName, value
}
