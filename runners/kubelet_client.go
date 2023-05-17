package runners

import (
	"bytes"
	"context"
	"fmt"
	"net/url"

	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type kubeletClient struct {
}

func (c *kubeletClient) GetMetrics(ctx context.Context) (map[types.NamespacedName]*VolumeStats, error) {
	// create a Kubernetes client using in-cluster configuration
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// get a list of nodes and IP addresses
	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// create a map to hold PVC usage data
	pvcUsage := make(map[types.NamespacedName]*VolumeStats)

	// use an errgroup to query kubelet for PVC usage on each node
	eg, ctx := errgroup.WithContext(ctx)
	for _, node := range nodes.Items {
		nodeName := node.Name
		eg.Go(func() error {
			return getPVCUsage(clientset, nodeName, pvcUsage, ctx)
		})
	}

	// wait for all queries to complete and handle any errors
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	// print the PVC usage data
	/*for pvc, usage := range pvcUsage {
		fmt.Printf("%s: %s\n", pvc, usage)
	}
	*/
	return pvcUsage, nil
}

func getPVCUsage(clientset *kubernetes.Clientset, nodeName string, pvcUsage map[types.NamespacedName]*VolumeStats, ctx context.Context) error {
	proxyURL := fmt.Sprintf("/api/v1/nodes/%s/proxy/metrics", nodeName)
	reqURL, err := url.Parse(proxyURL)
	if err != nil {
		return errors.Wrap(err, "failed to create HTTP request")
	}
	req := clientset.CoreV1().RESTClient().Get().
		Namespace("").
		Resource("nodes").
		Name(nodeName).
		SubResource(reqURL.String()).
		Param("timeout", "5s")

	// make the request to the kubelet and handle the response
	resp := req.Do(ctx)

	if resp.Error() != nil {
		return errors.Errorf("failed to get stats from kubelet on node %s: HTTP status code %d", nodeName, resp.StatusCode)
	} else {
		parser := expfmt.TextParser{}
		respBody, err := resp.Raw()
		if err != nil {
			return errors.Wrapf(err, "failed to get kubelet response on node %s", nodeName)
		}
		metricFamilies, err := parser.TextToMetricFamilies(bytes.NewReader(respBody))
		if err != nil {
			return errors.Wrapf(err, "failed to read response body from kubelet on node %s", nodeName)
		}

		//volumeAvailableQuery
		if gauge, ok := metricFamilies[volumeAvailableQuery]; ok {
			for _, m := range gauge.Metric {
				pvcName, value := parseMetric(m)
				pvcUsage[pvcName] = new(VolumeStats)
				pvcUsage[pvcName].AvailableBytes = int64(value)
			}
		}
		//volumeCapacityQuery
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

		return nil
	}
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
