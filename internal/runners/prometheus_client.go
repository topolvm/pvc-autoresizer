package runners

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/topolvm/pvc-autoresizer/internal/metrics"
	"k8s.io/apimachinery/pkg/types"
)

// NewPrometheusClient returns a new prometheusClient.
// labelSelector is an optional comma-separated list of PromQL label matchers
// (e.g. `cluster="prod",env!="dev"`) appended to every volume stats query.
// This is useful when the Prometheus-compatible endpoint aggregates metrics
// from multiple clusters: without a selector, same-named PVCs from another
// cluster would be indistinguishable because results are keyed by namespace
// and PVC name only.
func NewPrometheusClient(url string, labelSelector string) (MetricsClient, error) {
	client, err := api.NewClient(api.Config{
		Address: url,
	})
	if err != nil {
		return nil, err
	}
	v1api := prometheusv1.NewAPI(client)

	return &prometheusClient{
		prometheusAPI: v1api,
		labelSelector: labelSelector,
	}, nil
}

type prometheusClient struct {
	prometheusAPI prometheusv1.API
	labelSelector string
}

// GetMetrics implements MetricsClient.GetMetrics
func (c *prometheusClient) GetMetrics(ctx context.Context) (map[types.NamespacedName]*VolumeStats, error) {
	volumeStatsMap := make(map[types.NamespacedName]*VolumeStats)

	availableBytes, err := c.getMetricValues(ctx, c.buildQuery(volumeAvailableQuery))
	if err != nil {
		return nil, err
	}

	capacityBytes, err := c.getMetricValues(ctx, c.buildQuery(volumeCapacityQuery))
	if err != nil {
		return nil, err
	}

	availableInodeSize, err := c.getMetricValues(ctx, c.buildQuery(inodesAvailableQuery))
	if err != nil {
		return nil, err
	}

	capacityInodeSize, err := c.getMetricValues(ctx, c.buildQuery(inodesCapacityQuery))
	if err != nil {
		return nil, err
	}

	for key, val := range availableBytes {
		vs := &VolumeStats{AvailableBytes: val}
		if cb, ok := capacityBytes[key]; ok {
			vs.CapacityBytes = cb
		} else {
			continue
		}
		if ais, ok := availableInodeSize[key]; ok {
			vs.AvailableInodeSize = ais
		} else {
			continue
		}
		if cis, ok := capacityInodeSize[key]; ok {
			vs.CapacityInodeSize = cis
		} else {
			continue
		}
		volumeStatsMap[key] = vs
	}

	return volumeStatsMap, nil
}

// buildQuery returns the PromQL query for a metric name, applying the
// configured label selector if one is set.
func (c *prometheusClient) buildQuery(metricName string) string {
	if c.labelSelector == "" {
		return metricName
	}
	return fmt.Sprintf("%s{%s}", metricName, c.labelSelector)
}

func (c *prometheusClient) getMetricValues(ctx context.Context, query string) (map[types.NamespacedName]int64, error) {
	res, _, err := c.prometheusAPI.Query(ctx, query, time.Now())
	if err != nil {
		metrics.MetricsClientFailTotal.Increment()
		return nil, err
	}

	if res.Type() != model.ValVector {
		return nil, fmt.Errorf("unknown response type: %s", res.Type().String())
	}
	resultMap := make(map[types.NamespacedName]int64)
	vec := res.(model.Vector)
	for _, val := range vec {
		nn := types.NamespacedName{
			Namespace: string(val.Metric["namespace"]),
			Name:      string(val.Metric["persistentvolumeclaim"]),
		}
		resultMap[nn] = int64(val.Value)
	}
	return resultMap, nil
}
