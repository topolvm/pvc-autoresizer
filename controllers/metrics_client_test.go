package controllers

import "context"

type prometheusClientMock struct{}

func (c *prometheusClientMock) GetMetrics(ctx context.Context, namespace, name string) (*VolumeStats, error) {
	return nil, nil
}
