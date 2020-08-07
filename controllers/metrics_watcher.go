package controllers

import (
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
	// ctx := contextFromStopChannel(ch)

	defer ticker.Stop()
	for {
		select {
		case <-ch:
			return nil
		case <-ticker.C:
			// r.prometheusAPI.Query(ctx, query string, )
		}
	}
}
