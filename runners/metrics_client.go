package runners

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"k8s.io/apimachinery/pkg/types"
)

var errNotFound = errors.New("metrics not found")

const (
	volumeUsedQuery      = "kubelet_volume_stats_used_bytes"
	volumeAvailableQuery = "kubelet_volume_stats_available_bytes"
	volumeCapacityQuery  = "kubelet_volume_stats_capacity_bytes"
)

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

type MetricsClient interface {
	GetMetrics(ctx context.Context) (map[types.NamespacedName]*VolumeStats, error)
}

type VolumeStats struct {
	AvailableBytes int64
	UsedBytes      int64
	CapacityBytes  int64
}

type prometheusClient struct {
	prometheusAPI prometheusv1.API
}

func (c *prometheusClient) GetMetrics(ctx context.Context) (map[types.NamespacedName]*VolumeStats, error) {
	volumeStatsMap := make(map[types.NamespacedName]*VolumeStats)
	var err error

	usedBytes, err := c.getMetricValues(ctx, volumeUsedQuery)
	if err != nil {
		return nil, err
	}
	fmt.Printf("[DEBUG] usedBytes is %v\n", usedBytes)

	availableBytes, err := c.getMetricValues(ctx, volumeAvailableQuery)
	if err != nil {
		return nil, err
	}
	fmt.Printf("[DEBUG] availableBytes is %v=\n", availableBytes)

	capacityBytes, err := c.getMetricValues(ctx, volumeCapacityQuery)
	if err != nil {
		return nil, err
	}
	fmt.Printf("[DEBUG] capacityBytes is %v\n", capacityBytes)

	for key, val := range usedBytes {
		if _, ok := availableBytes[key]; !ok {
			continue
		}
		if _, ok := capacityBytes[key]; !ok {
			continue
		}

		vs := VolumeStats{
			AvailableBytes: availableBytes[key],
			UsedBytes:      val,
			CapacityBytes:  capacityBytes[key],
		}
		volumeStatsMap[key] = &vs
	}

	return volumeStatsMap, nil
}

func (c *prometheusClient) getMetricValues(ctx context.Context, query string) (map[types.NamespacedName]int64, error) {
	res, _, err := c.prometheusAPI.Query(ctx, query, time.Now())
	if err != nil {
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
