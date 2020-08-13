package runners

import (
	"context"
	"github.com/go-logr/logr"
	"strconv"
	"time"

	"github.com/cybozu-go/log"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func newPVCAutoresizer(interval time.Duration) *pvcAutoresizer {
	ch := make(chan event.GenericEvent)

	return &pvcAutoresizer{
		channel:  ch,
		interval: interval,
	}
}

func (w *pvcAutoresizer) InjectClient(c client.Client) error {
	w.client = c
	return nil
}

type pvcAutoresizer struct {
	channel       chan event.GenericEvent
	client        client.Client
	metricsClient MetricsClient
	interval      time.Duration
}

func (w *pvcAutoresizer) Start(ch <-chan struct{}) error {
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

func (w *pvcAutoresizer) getStorageClassList(ctx context.Context) (*storagev1.StorageClassList, error) {
	var scs storagev1.StorageClassList
	err := w.client.List(ctx, &scs, client.MatchingFields(map[string]string{resizeEnableIndexKey: "true"}))
	if err != nil {
		return nil, err
	}
	return &scs, nil
}

func (w *pvcAutoresizer) notifyPVCEvent(ctx context.Context) error {
	scs, err := w.getStorageClassList(ctx)
	if err != nil {
		return err
	}

	vsMap, err := w.metricsClient.GetMetrics(ctx)
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
			// TODO reconcile
			resize(ctx,TODO)
		}
	}

	return ctrl.Result{}, nil
}

func resize(ctx context.Context, pvc *corev1.PersistentVolumeClaim, vs *VolumeStats) error {

	threshold, err := convertSizeInBytes(pvc.Annotations[ResizeThresholdAnnotation], vs.CapacityBytes, DefaultThreshold)
	if err != nil {
		return err
	}
	increase, err := convertSizeInBytes(pvc.Annotations[ResizeIncreaseAnnotation], pvc.Spec.Resources.Limits.Storage().Value(), DefaultIncrease)
	if err != nil {
		return err
	}
	preCap, exist := pvc.Annotations[PreviousCapacityBytesAnnotation]
	if exist {
		preCapInt64, err := strconv.ParseInt(preCap, 10, 64)
		if err != nil {
			return ctrl.Result{}, err
		}
		if preCapInt64 == vs.CapacityBytes {
			log.Info("waiting for resizing...", "capacity", vs.CapacityBytes)
			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: 30 * time.Second,
			}, nil
		}
	}

	if threshold > vs.AvailableBytes {
		if pvc.Annotations == nil {
			pvc.Annotations = make(map[string]string)
		}
		curReq := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
		newReq := resource.NewQuantity(curReq.Value()+increase, resource.BinarySI)
		limitRes := pvc.Spec.Resources.Limits[corev1.ResourceStorage]
		if curReq.Cmp(limitRes) == 0 {
			return ctrl.Result{}, nil
		}
		if newReq.Cmp(limitRes) > 0 {
			newReq = &limitRes
		}

		pvc.Spec.Resources.Requests[corev1.ResourceStorage] = *newReq
		pvc.Annotations[PreviousCapacityBytesAnnotation] = strconv.FormatInt(vs.CapacityBytes, 10)
		err = r.Client.Update(ctx, &pvc)
		if err != nil {
			return ctrl.Result{}, err
		}
		log.Info("resize started", "new capacity", newReq.Value())
		r.Recorder.Eventf(&pvc, corev1.EventTypeNormal, "Resized", "PVC volume is resized to %s", newReq.String())
	}
}
