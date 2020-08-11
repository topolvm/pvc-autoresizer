package controllers

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func newPVCWatcher(interval time.Duration) *pvcWatcher {
	ch := make(chan event.GenericEvent)

	return &pvcWatcher{
		channel:  ch,
		interval: interval,
	}
}

func (w *pvcWatcher) InjectClient(c client.Client) error {
	w.client = c
	return nil
}

type pvcWatcher struct {
	channel  chan event.GenericEvent
	client   client.Client
	interval time.Duration
}

func (w *pvcWatcher) Start(ch <-chan struct{}) error {
	ticker := time.NewTicker(w.interval)
	ctx := context.Background()

	defer ticker.Stop()
	for {
		select {
		case <-ch:
			return nil
		case <-ticker.C:
			err := w.notifyPVCEvent(ctx)
			if err != nil {
				return err
			}
		}
	}
}

func isTargetPVC(pvc *corev1.PersistentVolumeClaim) bool {
	if pvc.Spec.Resources.Limits.Storage() == nil {
		return false
	}
	if pvc.Spec.VolumeMode != nil && *pvc.Spec.VolumeMode != corev1.PersistentVolumeFilesystem {
		return false
	}
	return true
}

func (w *pvcWatcher) getStorageClassList(ctx context.Context) (*storagev1.StorageClassList, error) {
	var scs storagev1.StorageClassList
	err := w.client.List(ctx, &scs, client.MatchingFields(map[string]string{resizeEnableIndexKey: "true"}))
	if err != nil {
		return nil, err
	}
	return &scs, nil
}

func (w *pvcWatcher) notifyPVCEvent(ctx context.Context) error {
	scs, err := w.getStorageClassList(ctx)
	if err != nil {
		return err
	}

	for _, sc := range scs.Items {
		var pvcs corev1.PersistentVolumeClaimList
		err = w.client.List(ctx, &pvcs, client.MatchingFields(map[string]string{storageClassNameIndexKey: sc.Name}))
		if err != nil {
			return err
		}
		for _, pvc := range pvcs.Items {
			if !isTargetPVC(&pvc) {
				continue
			}
			w.channel <- event.GenericEvent{
				Meta: &metav1.ObjectMeta{
					Name:      pvc.Name,
					Namespace: pvc.Namespace,
				},
			}
		}
	}
	return nil
}
