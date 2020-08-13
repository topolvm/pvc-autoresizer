package runners

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
)

type prometheusClientMock struct {
	stats map[types.NamespacedName]*VolumeStats
}

func (c *prometheusClientMock) GetMetrics(ctx context.Context) (map[types.NamespacedName]*VolumeStats, error) {
	return c.stats, nil
}

func (c *prometheusClientMock) addResponse(key types.NamespacedName, stats *VolumeStats) {
	if c.stats == nil {
		c.stats = make(map[types.NamespacedName]*VolumeStats)
	}
	c.stats[key] = stats
}
