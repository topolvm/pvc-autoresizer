package controllers

import (
	"context"
	"errors"
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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const resizeEnableIndexKey = ".metadata.annotations[resize.topolvm.io/enabled]"
const storageClassNameIndexKey = ".spec.storageClassName"

// PersistentVolumeClaimReconciler reconciles a PersistentVolumeClaim object
type PersistentVolumeClaimReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	MetricsClient
}

// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch

func (r *PersistentVolumeClaimReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("persistentvolumeclaim", req.NamespacedName)

	// your logic here
	var pvc corev1.PersistentVolumeClaim
	err := r.Get(ctx, req.NamespacedName, &pvc)
	if err != nil {
		return ctrl.Result{}, err
	}

	namespace, name := req.NamespacedName.Namespace, req.NamespacedName.Name
	vs, err := r.MetricsClient.GetMetrics(ctx, namespace, name)
	if errors.Is(err, errNotFound) {
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: 30 * time.Second,
		}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	threshold, err := convertSizeInBytes(pvc.Annotations[ResizeThresholdAnnotation], vs.CapacityBytes, DefaultThreshold)
	if err != nil {
		return ctrl.Result{}, err
	}
	increase, err := convertSizeInBytes(pvc.Annotations[ResizeIncreaseAnnotation], pvc.Spec.Resources.Limits.Storage().Value(), DefaultIncrease)
	if err != nil {
		return ctrl.Result{}, err
	}

	preCap, exist := pvc.Annotations[PreviousCapacityBytesAnnotation]
	if exist {
		preCapInt64, err := strconv.ParseInt(preCap, 10, 64)
		if err != nil {
			return ctrl.Result{}, err
		}
		if preCapInt64 == vs.CapacityBytes {
			log.Info("waiting for resizing...", "namespace", req.Namespace, "name", req.Name, "capacity", vs.CapacityBytes)
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
		log.Info("resize started", "namespace", req.Namespace, "name", req.Name, "new capacity", newReq.Value())
	}

	return ctrl.Result{}, nil
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

func (r *PersistentVolumeClaimReconciler) SetupWithManager(mgr ctrl.Manager, interval time.Duration) error {
	pred := predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return true },
		DeleteFunc:  func(event.DeleteEvent) bool { return false },
		UpdateFunc:  func(e event.UpdateEvent) bool { return true },
		GenericFunc: func(event.GenericEvent) bool { return true },
	}

	err := mgr.GetFieldIndexer().IndexField(context.Background(), &storagev1.StorageClass{}, resizeEnableIndexKey, indexByResizeEnableAnnotation)
	if err != nil {
		return err
	}

	err = mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.PersistentVolumeClaim{}, storageClassNameIndexKey, indexByStorageClassName)
	if err != nil {
		return err
	}

	external := newPVCWatcher(interval)
	err = mgr.Add(external)
	if err != nil {
		return err
	}
	src := source.Channel{
		Source: external.channel,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.PersistentVolumeClaim{}).
		Watches(&src, &handler.EnqueueRequestForObject{}).
		WithEventFilter(pred).
		Complete(r)
}
