package runners

import (
	"context"
	"sync"

	"k8s.io/apimachinery/pkg/types"
)

type prometheusClientMock struct {
	stats map[types.NamespacedName]*VolumeStats
	mutex sync.Mutex
}

func (c *prometheusClientMock) GetMetrics(ctx context.Context) (map[types.NamespacedName]*VolumeStats, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	copied := make(map[types.NamespacedName]*VolumeStats)
	for k, v := range c.stats {
		copied[k] = v
	}
	return copied, nil
}

func (c *prometheusClientMock) setResponce(key types.NamespacedName, stats *VolumeStats) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.stats == nil {
		c.stats = make(map[types.NamespacedName]*VolumeStats)
	}
	c.stats[key] = stats
}
