package runners

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/topolvm/pvc-autoresizer/internal/metrics"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const nodeMetricsRequestTimeout = 10 * time.Second

// NewK8sMetricsApiClient returns a new k8sMetricsApiClient client
func NewK8sMetricsApiClient(log logr.Logger) (MetricsClient, error) {
	return &k8sMetricsApiClient{log: log}, nil
}

type k8sMetricsApiClient struct {
	log logr.Logger
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

	nodeNames := make([]string, len(nodes.Items))
	for i, node := range nodes.Items {
		nodeNames[i] = node.Name
	}

	return gatherFromNodes(ctx, c.log, nodeNames, func(ctx context.Context, nodeName string) (map[types.NamespacedName]*VolumeStats, error) {
		return getPVCUsageFromK8sMetricsAPI(ctx, clientset, nodeName)
	})
}

// gatherFromNodes queries each node independently and merges the successful results.
// It returns an error only when every node fails.
func gatherFromNodes(
	ctx context.Context,
	log logr.Logger,
	nodeNames []string,
	fetch func(ctx context.Context, nodeName string) (map[types.NamespacedName]*VolumeStats, error),
) (map[types.NamespacedName]*VolumeStats, error) {
	pvcUsage := make(map[types.NamespacedName]*VolumeStats)
	failedCount := 0
	var mu sync.Mutex // serialize writes to pvcUsage and failedCount

	var wg sync.WaitGroup
	for _, nodeName := range nodeNames {
		wg.Go(func() {
			nodeCtx, cancel := context.WithTimeout(ctx, nodeMetricsRequestTimeout)
			defer cancel()
			nodePVCUsage, err := fetch(nodeCtx, nodeName)
			if err != nil {
				if ctx.Err() != nil {
					// caller is shutting down, not a node failure
					return
				}
				log.Error(err, "failed to get volume stats from node, skipping", "node", nodeName)
				mu.Lock()
				failedCount++
				mu.Unlock()
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for k, v := range nodePVCUsage {
				pvcUsage[k] = v
			}
		})
	}
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if failedCount > 0 {
		metrics.MetricsClientFailTotal.Increment()
	}
	if len(nodeNames) > 0 && failedCount == len(nodeNames) {
		return nil, fmt.Errorf("failed to get volume stats from all %d nodes", len(nodeNames))
	}
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
