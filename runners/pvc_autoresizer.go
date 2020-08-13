package runners

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch

const resizeEnableIndexKey = ".metadata.annotations[resize.topolvm.io/enabled]"
const storageClassNameIndexKey = ".spec.storageClassName"

func NewPVCAutoresizer(mc MetricsClient, interval time.Duration, recorder record.EventRecorder) *pvcAutoresizer {

	return &pvcAutoresizer{
		metricsClient: mc,
		interval:      interval,
		recorder:      recorder,
	}
}

func (w *pvcAutoresizer) InjectClient(c client.Client) error {
	w.client = c
	return nil
}

func (w *pvcAutoresizer) InjectLogger(log logr.Logger) error {
	w.log = log
	return nil
}

type pvcAutoresizer struct {
	client        client.Client
	metricsClient MetricsClient
	interval      time.Duration
	log           logr.Logger
	recorder      record.EventRecorder
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
	if pvc.Spec.Resources.Limits.Storage().IsZero() {
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
			namespacedName := types.NamespacedName{
				Namespace: pvc.Namespace,
				Name:      pvc.Name,
			}
			if _, ok := vsMap[namespacedName]; !ok {
				continue
			}
			err = w.resize(ctx, &pvc, vsMap[namespacedName])
			if err != nil {
				// TODO
				return err
			}
		}
	}

	return nil
}

func (w *pvcAutoresizer) resize(ctx context.Context, pvc *corev1.PersistentVolumeClaim, vs *VolumeStats) error {
	log := w.log.WithName("resize").WithValues("namespace", pvc.Namespace, "name", pvc.Name)

	threshold, err := convertSizeInBytes(pvc.Annotations[ResizeThresholdAnnotation], vs.CapacityBytes, DefaultThreshold)
	if err != nil {
		return err
	}
	fmt.Printf("[DEBUG] threshold is %d\n", threshold)

	increase, err := convertSizeInBytes(pvc.Annotations[ResizeIncreaseAnnotation], pvc.Spec.Resources.Limits.Storage().Value(), DefaultIncrease)
	if err != nil {
		return err
	}
	fmt.Printf("[DEBUG] increase is %d\n", increase)

	preCap, exist := pvc.Annotations[PreviousCapacityBytesAnnotation]
	if exist {
		preCapInt64, err := strconv.ParseInt(preCap, 10, 64)
		if err != nil {
			return err
		}
		if preCapInt64 == vs.CapacityBytes {
			log.Info("waiting for resizing...", "capacity", vs.CapacityBytes)
			return nil
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
			return nil
		}
		if newReq.Cmp(limitRes) > 0 {
			newReq = &limitRes
		}

		pvc.Spec.Resources.Requests[corev1.ResourceStorage] = *newReq
		pvc.Annotations[PreviousCapacityBytesAnnotation] = strconv.FormatInt(vs.CapacityBytes, 10)
		err = w.client.Update(ctx, pvc)
		if err != nil {
			return err
		}
		log.Info("resize started", "current caapcity", curReq.Value(), "new capacity", newReq.Value())
		w.recorder.Eventf(pvc, corev1.EventTypeNormal, "Resized", "PVC volume is resized to %s", newReq.String())
	}

	return nil
}

func indexByResizeEnableAnnotation(obj runtime.Object) []string {
	sc := obj.(*storagev1.StorageClass)
	if val, ok := sc.Annotations[AutoResizeEnabledKey]; ok {
		return []string{val}
	}

	return []string{}
}

func indexByStorageClassName(obj runtime.Object) []string {
	pvc := obj.(*corev1.PersistentVolumeClaim)
	scName := pvc.Spec.StorageClassName
	if scName == nil {
		return []string{}
	}
	return []string{*scName}
}

func (w *pvcAutoresizer) SetupWithManager(mgr ctrl.Manager) error {
	err := mgr.GetFieldIndexer().IndexField(context.Background(), &storagev1.StorageClass{}, resizeEnableIndexKey, indexByResizeEnableAnnotation)
	if err != nil {
		return err
	}

	err = mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.PersistentVolumeClaim{}, storageClassNameIndexKey, indexByStorageClassName)
	if err != nil {
		return err
	}

	return nil
}

func convertSizeInBytes(valStr string, capacity int64, defaultVal string) (int64, error) {
	if len(valStr) == 0 {
		valStr = defaultVal
	}

	if strings.HasSuffix(valStr, "%") {
		rate, err := strconv.ParseFloat(strings.TrimRight(valStr, "%"), 64)
		if err != nil {
			return 0, err
		}
		if rate < 0.0 || 100.0 < rate {
			return 0, fmt.Errorf("annotation value should be between 0%% to 100%%: %s", valStr)
		}

		// rounding up the result to Gi
		res := int64(math.Ceil(float64(capacity)*rate/100.0/(1<<30))) << 30
		return res, nil
	}

	quantity, err := resource.ParseQuantity(valStr)
	if err != nil {
		return 0, err
	}
	val := quantity.Value()
	if val < 0 || capacity < val {
		return 0, fmt.Errorf("annotation value should be between 0 to capacity value(%d): %s", capacity, valStr)
	}
	return val, nil
}
