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

// NewPrometheusClient returns a new prometheusClient
func NewPrometheusClient(url string) (MetricsClient, error) {
	client, err := api.NewClient(api.Config{
		Address: url,
	})
	if err != nil {
		return nil, err
	}
	v1api := prometheusv1.NewAPI(client)

	return &prometheusClient{
		prometheusAPI: v1api,
	}, nil
}

type prometheusClient struct {
	prometheusAPI prometheusv1.API
}

// GetMetrics implements MetricsClient.GetMetrics
func (c *prometheusClient) GetMetrics(ctx context.Context) (map[types.NamespacedName]*VolumeStats, error) {
	volumeStatsMap := make(map[types.NamespacedName]*VolumeStats)

	availableBytes, err := c.getMetricValues(ctx, volumeAvailableQuery)
	if err != nil {
		return nil, err
	}

	capacityBytes, err := c.getMetricValues(ctx, volumeCapacityQuery)
	if err != nil {
		return nil, err
	}

	availableInodeSize, err := c.getMetricValues(ctx, inodesAvailableQuery)
	if err != nil {
		return nil, err
	}

	capacityInodeSize, err := c.getMetricValues(ctx, inodesCapacityQuery)
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
