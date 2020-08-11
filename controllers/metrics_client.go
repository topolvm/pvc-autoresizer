package controllers

import (
	"context"
	"log"
	"time"

	"github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

func newPrometheusClient(url string) (metricsClient, error) {

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

type metricsClient interface {
	GetMetrics(context.Context, string, string) VolumeStats
}

type VolumeStats struct {
	AvailableBytes int64
	UsedBytes      int64
	CapacityBytes  int64
}

type prometheusClient struct {
	prometheusAPI prometheusv1.API
}

func (c *prometheusClient) GetMetrics(ctx context.Context, namespace, name string) VolumeStats {
	q := "kubelet_volume_stats_used_bytes"
	res, _, err := c.prometheusAPI.Query(ctx, q, time.Now())
	if err != nil {
		log.Fatalln(err)
	}

	if res.Type() == model.ValVector {
		v := res.(model.Vector)
		for _, val := range v {
			if string(val.Metric["namespace"]) != namespace || string(val.Metric["persistentvolumeclaim"]) != name {
				continue
			}
		}
	}
}
