package controllers

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func newPVCWatcher(url string) (*pvcWatcher, error) {
	ch := make(chan event.GenericEvent)

	return &pvcWatcher{
		channel: ch,
	}, nil
}

func (w *pvcWatcher) InjectClient(c client.Client) error {
	w.client = c
	return nil
}

type pvcWatcher struct {
	channel chan event.GenericEvent
	client  client.Client
}

func (w *pvcWatcher) Start(ch <-chan struct{}) error {
	ticker := time.NewTicker(10 * time.Second)
	_ = context.Background()

	defer ticker.Stop()
	for {
		select {
		case <-ch:
			return nil
		case <-ticker.C:

		}
	}
}

func filterPVC(pvc *corev1.PersistentVolumeClaim) bool {
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
	_, err := w.getStorageClassList(ctx)
	if err != nil {
		return err
	}

	return nil

	// var pvcs corev1.PersistentVolumeClaimList
	// err := w.client.List(ctx, &pvcs)
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }
	// if pvc.Spec.StorageClassName == nil {
	// 	return ctrl.Result{}, errors.New("`pvc.spec.StorageClassName` should not be empty")
	// }
}
