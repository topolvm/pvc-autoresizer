package runners

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
)

func TestGatherFromNodes(t *testing.T) {
	pvcA := types.NamespacedName{Namespace: "ns", Name: "pvc-a"}
	pvcB := types.NamespacedName{Namespace: "ns", Name: "pvc-b"}

	t.Run("merges results from all nodes when every node succeeds", func(t *testing.T) {
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
		if len(got) != 2 {
			t.Fatalf("expected 2 PVCs, got %d: %+v", len(got), got)
		}
		if got[pvcA] == nil || got[pvcA].AvailableBytes != 1 {
			t.Errorf("pvc-a missing or wrong: %+v", got[pvcA])
		}
		if got[pvcB] == nil || got[pvcB].AvailableBytes != 2 {
			t.Errorf("pvc-b missing or wrong: %+v", got[pvcB])
		}
	})

	t.Run("a failing node only loses its own data", func(t *testing.T) {
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
		if len(got) != 1 {
			t.Fatalf("expected 1 PVC (from the healthy node only), got %d: %+v", len(got), got)
		}
		if got[pvcA] == nil {
			t.Errorf("expected pvc-a from the healthy node, got %+v", got)
		}
	})

	t.Run("returns an error when every node fails", func(t *testing.T) {
		fetch := func(_ context.Context, nodeName string) (map[types.NamespacedName]*VolumeStats, error) {
			return nil, errors.New("kubelet unreachable")
		}

		_, err := gatherFromNodes(context.Background(), logr.Discard(), []string{"node-a", "node-b"}, fetch)

		if err == nil {
			t.Fatal("expected an error when every node fails, got nil")
		}
	})

	t.Run("a cancelled parent context is not counted as a node failure", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		fetch := func(ctx context.Context, _ string) (map[types.NamespacedName]*VolumeStats, error) {
			return nil, ctx.Err()
		}

		// This only verifies gatherFromNodes does not panic or hang; the log/metric
		// suppression itself is not directly observable from this test.
		_, err := gatherFromNodes(ctx, logr.Discard(), []string{"node-a", "node-b"}, fetch)
		if err == nil {
			t.Fatal("expected an error since no node succeeded, got nil")
		}
	})

	t.Run("each node's fetch gets a bounded context", func(t *testing.T) {
		fetch := func(ctx context.Context, _ string) (map[types.NamespacedName]*VolumeStats, error) {
			if _, ok := ctx.Deadline(); !ok {
				t.Error("expected fetch to receive a context with a deadline")
			}
			return map[types.NamespacedName]*VolumeStats{}, nil
		}

		if _, err := gatherFromNodes(context.Background(), logr.Discard(), []string{"node-a"}, fetch); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("no nodes returns an empty, non-nil map", func(t *testing.T) {
		fetch := func(_ context.Context, nodeName string) (map[types.NamespacedName]*VolumeStats, error) {
			t.Error("fetch should not be called with no nodes")
			return nil, nil
		}

		got, err := gatherFromNodes(context.Background(), logr.Discard(), nil, fetch)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected a non-nil map")
		}
		if len(got) != 0 {
			t.Errorf("expected an empty map, got %+v", got)
		}
	})
}
