package controllers

import (
	"context"
	"errors"
	storagev1 "k8s.io/api/storage/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PersistentVolumeClaimReconciler reconciles a PersistentVolumeClaim object
type PersistentVolumeClaimReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch

func (r *PersistentVolumeClaimReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("persistentvolumeclaim", req.NamespacedName)

	// your logic here
	var pvc corev1.PersistentVolumeClaim
	err := r.Get(ctx, req.NamespacedName, &pvc)
	if err != nil {
		return ctrl.Result{}, err
	}
	if pvc.Spec.StorageClassName == nil {
		return ctrl.Result{}, errors.New("`pvc.spec.StorageClassName` should not be empty")
	}

	var sc storagev1.StorageClass
	err = r.Get(ctx, client.ObjectKey{Name: *pvc.Spec.StorageClassName}, &sc)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PersistentVolumeClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			pvc := e.Object.(*corev1.PersistentVolumeClaim)
			return filterPVC(pvc)
		},
		DeleteFunc: func(event.DeleteEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			pvc := e.ObjectNew.(*corev1.PersistentVolumeClaim)
			return filterPVC(pvc)
		},
		GenericFunc: func(event.GenericEvent) bool { return true },
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.PersistentVolumeClaim{}).
		WithEventFilter(pred).
		Complete(r)
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
