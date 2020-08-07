package controllers

import (
	"context"
	"github.com/prometheus/common/model"
	"time"

	"github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func newMetricsWatcher(url string) (*metricsWatcher, error) {
	ch := make(chan event.GenericEvent)

	client, err := api.NewClient(api.Config{
		Address: url,
	})
	if err != nil {
		return nil, err
	}
	v1api := prometheusv1.NewAPI(client)

	return &metricsWatcher{
		channel:       ch,
		prometheusAPI: v1api,
	}, nil
}

func (r *metricsWatcher) InjectClient(c client.Client) error {
	r.client = c
	return nil
}

type metricsWatcher struct {
	channel       chan event.GenericEvent
	client        client.Client
	prometheusAPI prometheusv1.API
}

func (r metricsWatcher) Start(ch <-chan struct{}) error {
	ticker := time.NewTicker(10 * time.Second)
	ctx := context.Background()
	q := "kubelet_volume_stats_used_bytes"

	defer ticker.Stop()
	for {
		select {
		case <-ch:
			return nil
		case <-ticker.C:
			res, _, err := r.prometheusAPI.Query(ctx, q, time.Now())
			if err != nil {
				return err
			}
			if res.Type() == model.ValVector {
				v := res.(model.Vector)
			}
		}
	}
}
