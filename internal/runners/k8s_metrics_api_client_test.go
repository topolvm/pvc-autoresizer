package runners

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
)

func TestGatherFromNodesAllSucceed(t *testing.T) {
	pvcA := types.NamespacedName{Namespace: "ns", Name: "pvc-a"}
	pvcB := types.NamespacedName{Namespace: "ns", Name: "pvc-b"}

	fetch := func(_ context.Context, nodeName string) (map[types.NamespacedName]*VolumeStats, error) {
		switch nodeName {
		case "node-a":
			return map[types.NamespacedName]*VolumeStats{pvcA: {AvailableBytes: 1}}, nil
		case "node-b":
			return map[types.NamespacedName]*VolumeStats{pvcB: {AvailableBytes: 2}}, nil
		default:
			t.Errorf("unexpected node: %s", nodeName)
			return nil, nil
		}
	}

	got, err := gatherFromNodes(context.Background(), logr.Discard(), []string{"node-a", "node-b"}, fetch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[pvcA] == nil || got[pvcA].AvailableBytes != 1 || got[pvcB] == nil || got[pvcB].AvailableBytes != 2 {
		t.Errorf("expected both PVCs merged, got %+v", got)
	}
}

func TestGatherFromNodesOneFails(t *testing.T) {
	pvcA := types.NamespacedName{Namespace: "ns", Name: "pvc-a"}

	fetch := func(_ context.Context, nodeName string) (map[types.NamespacedName]*VolumeStats, error) {
		if nodeName == "node-bad" {
			return nil, errors.New("kubelet unreachable")
		}
		return map[types.NamespacedName]*VolumeStats{pvcA: {AvailableBytes: 1}}, nil
	}

	got, err := gatherFromNodes(context.Background(), logr.Discard(), []string{"node-good", "node-bad"}, fetch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[pvcA] == nil {
		t.Errorf("expected only the healthy node's data, got %+v", got)
	}
}

func TestGatherFromNodesAllFail(t *testing.T) {
	fetch := func(_ context.Context, nodeName string) (map[types.NamespacedName]*VolumeStats, error) {
		return nil, errors.New("kubelet unreachable")
	}

	if _, err := gatherFromNodes(context.Background(), logr.Discard(), []string{"node-a", "node-b"}, fetch); err == nil {
		t.Fatal("expected an error when every node fails, got nil")
	}
}

func TestGatherFromNodesContextCancelled(t *testing.T) {
	fetch := func(_ context.Context, nodeName string) (map[types.NamespacedName]*VolumeStats, error) {
		return nil, errors.New("kubelet unreachable")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := gatherFromNodes(ctx, logr.Discard(), []string{"node-a", "node-b"}, fetch); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}
