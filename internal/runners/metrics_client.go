package runners

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
)

const (
	volumeAvailableQuery = "kubelet_volume_stats_available_bytes"
	volumeCapacityQuery  = "kubelet_volume_stats_capacity_bytes"
	inodesAvailableQuery = "kubelet_volume_stats_inodes_free"
	inodesCapacityQuery  = "kubelet_volume_stats_inodes"
)

// MetricsClient is an interface for getting metrics
type MetricsClient interface {
	// GetMetrics returns volume stats metrics of PVCs
	//
	// The volume stats consist of available bytes, capacity bytes, available inodes and capacity
	// inodes. This method returns volume stats for a PVC only if all four metrics of the PVC was
	// retrieved from the metrics source.
	GetMetrics(ctx context.Context) (map[types.NamespacedName]*VolumeStats, error)
}

// VolumeStats is a struct containing metrics used by pvc-autoresizer
type VolumeStats struct {
	AvailableBytes     int64
	CapacityBytes      int64
	AvailableInodeSize int64
	CapacityInodeSize  int64
}
