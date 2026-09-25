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

// nodeMetricsRequestTimeout bounds a single node's kubelet-proxy request, so that one
// node accepting a connection but never responding cannot block metrics collection for
// the whole cluster.
const nodeMetricsRequestTimeout = 10 * time.Second

// NewK8sMetricsApiClient returns a new k8sMetricsApiClient client
func NewK8sMetricsApiClient(log logr.Logger) (MetricsClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &k8sMetricsApiClient{log: log, clientset: clientset}, nil
}

type k8sMetricsApiClient struct {
	log       logr.Logger
	clientset *kubernetes.Clientset
}

func (c *k8sMetricsApiClient) GetMetrics(ctx context.Context) (map[types.NamespacedName]*VolumeStats, error) {
	// get a list of nodes
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, v1.ListOptions{})
	if err != nil {
		metrics.MetricsClientFailTotal.Increment()
		return nil, err
	}

	nodeNames := make([]string, len(nodes.Items))
	for i, node := range nodes.Items {
		nodeNames[i] = node.Name
	}

	return gatherFromNodes(ctx, c.log, nodeNames, func(ctx context.Context, nodeName string) (map[types.NamespacedName]*VolumeStats, error) {
		return getPVCUsageFromK8sMetricsAPI(ctx, c.clientset, nodeName)
	})
}

// gatherFromNodes queries each node independently via fetch and merges the successful
// results. A node that fails to respond (e.g. mid-scale-down) only loses its own PVC
// data; it must not abort metrics collection for the rest of the cluster. It returns an
// error only if every node failed, so a total outage is still reported to the caller
// instead of looking like a cluster with no PVCs to resize.
func gatherFromNodes(
	ctx context.Context,
	log logr.Logger,
	nodeNames []string,
	fetch func(ctx context.Context, nodeName string) (map[types.NamespacedName]*VolumeStats, error),
) (map[types.NamespacedName]*VolumeStats, error) {
	pvcUsage := make(map[types.NamespacedName]*VolumeStats)
	successCount := 0
	var mu sync.Mutex // serializes writes to pvcUsage and successCount

	var wg sync.WaitGroup
	for _, nodeName := range nodeNames {
		wg.Add(1)
		go func() {
			defer wg.Done()
			nodeCtx, cancel := context.WithTimeout(ctx, nodeMetricsRequestTimeout)
			defer cancel()
			nodePVCUsage, err := fetch(nodeCtx, nodeName)
			if err != nil {
				if ctx.Err() != nil {
					// The caller is shutting down; this is not a node failure.
					return
				}
				log.Error(err, "failed to get volume stats from node, skipping", "node", nodeName)
				metrics.MetricsClientFailTotal.Increment()
				return
			}
			mu.Lock()
			defer mu.Unlock()
			successCount++
			for k, v := range nodePVCUsage {
				pvcUsage[k] = v
			}
		}()
	}
	wg.Wait()

	if len(nodeNames) > 0 && successCount == 0 {
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
