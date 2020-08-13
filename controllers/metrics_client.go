package controllers

import (
	"context"
	"errors"
	"time"

	"github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

var errNotFound = errors.New("metrics not found")

const (
	volumeUsedQuery      = "kubelet_volume_stats_used_bytes"
	volumeAvailableQuery = "kubelet_volume_stats_available_bytes"
	volumeCapacityQuery  = "kubelet_volume_stats_capacity_bytes"
)

type MetricsClient interface {
	GetMetrics(context.Context, string, string) (*VolumeStats, error)
}

type VolumeStats struct {
	AvailableBytes int64
	UsedBytes      int64
	CapacityBytes  int64
}

type prometheusClient struct {
	prometheusAPI prometheusv1.API
}

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

func (c *prometheusClient) GetMetrics(ctx context.Context, namespace, name string) (*VolumeStats, error) {
	volumeStats := VolumeStats{}
	var err error

	volumeStats.UsedBytes, err = c.getMetricValue(ctx, volumeUsedQuery, namespace, name)
	if err != nil {
		return nil, err
	}
	volumeStats.AvailableBytes, err = c.getMetricValue(ctx, volumeAvailableQuery, namespace, name)
	if err != nil {
		return nil, err
	}
	volumeStats.CapacityBytes, err = c.getMetricValue(ctx, volumeCapacityQuery, namespace, name)
	if err != nil {
		return nil, err
	}

	return &volumeStats, nil
}

func (c *prometheusClient) getMetricValue(ctx context.Context, query, namespace, name string) (int64, error) {
	res, _, err := c.prometheusAPI.Query(ctx, query, time.Now())
	if err != nil {
		return 0, err
	}

	if res.Type() == model.ValVector {
		v := res.(model.Vector)
		for _, val := range v {
			if string(val.Metric["namespace"]) != namespace || string(val.Metric["persistentvolumeclaim"]) != name {
				continue
			}
			return int64(val.Value), nil
		}
	}

	return 0, errNotFound
}
