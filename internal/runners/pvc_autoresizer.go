package runners

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	pvcautoresizer "github.com/topolvm/pvc-autoresizer"
	"github.com/topolvm/pvc-autoresizer/internal/metrics"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"github.com/topolvm/pvc-autoresizer/internal/notifications"
)

//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create

const resizeEnableIndexKey = ".metadata.annotations[resize.topolvm.io/enabled]"
const storageClassNameIndexKey = ".spec.storageClassName"
const logLevelWarn = 3
const limitWarningThresholdPercent = 80 // Threshold percentage for limit warning
const lastWarningTimestampAnnotation = "pvc-autoresizer.topolvm.io/last-warning-timestamp"
const lastLimitReachedTimestampAnnotation = "pvc-autoresizer.topolvm.io/last-limit-reached-timestamp"

// NewPVCAutoresizer returns a new pvcAutoresizer struct
func NewPVCAutoresizer(mc MetricsClient, c client.Client, log logr.Logger, interval time.Duration,
	recorder record.EventRecorder, slackConfig *notifications.SlackConfig) manager.Runnable {

	var slackNotifier *notifications.SlackNotifier
	if slackConfig != nil {
		slackNotifier = notifications.NewSlackNotifier(*slackConfig)
	}

	return &pvcAutoresizer{
		metricsClient: mc,
		client:        c,
		log:           log,
		interval:      interval,
		recorder:      recorder,
		slackNotifier: slackNotifier,
	}
}

type pvcAutoresizer struct {
	client        client.Client
	metricsClient MetricsClient
	interval      time.Duration
	log           logr.Logger
	recorder      record.EventRecorder
	slackNotifier *notifications.SlackNotifier
}

// Start implements manager.Runnable
func (w *pvcAutoresizer) Start(ctx context.Context) error {
	log := w.log.WithName("Start")
	log.Info("starting pvc autoresizer")

	// Send startup notification after leader election
	if w.slackNotifier != nil && !w.slackNotifier.IsStartupNotificationDisabled() {
		clusterName := os.Getenv("CLUSTER_NAME")
		if clusterName == "" {
			clusterName = "current cluster"
		}
		message := fmt.Sprintf("🚀 PVC Autoresizer started monitoring in %s\n"+
			"*Configuration:*\n"+
			"• Monitoring Interval: %s",
			clusterName,
			w.interval)
		if err := w.slackNotifier.SendStartupNotification(message); err != nil {
			log.Error(err, "failed to send startup notification")
		}
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			startTime := time.Now()
			if err := w.reconcile(ctx); err != nil {
				log.Error(err, "reconciliation failed")
			}
			metrics.ResizerLoopSecondsTotal.Add(time.Since(startTime).Seconds())
		}
	}
}

func isTargetPVC(pvc *corev1.PersistentVolumeClaim) (bool, error) {
	quantity, err := PvcStorageLimit(pvc)
	if err != nil {
		return false, fmt.Errorf("invalid storage limit: %w", err)
	}
	if quantity.IsZero() {
		return false, nil
	}
	if pvc.Spec.VolumeMode != nil && *pvc.Spec.VolumeMode != corev1.PersistentVolumeFilesystem {
		return false, nil
	}
	if pvc.Status.Phase != corev1.ClaimBound {
		return false, nil
	}
	return true, nil
}

func (w *pvcAutoresizer) getStorageClassList(ctx context.Context) (*storagev1.StorageClassList, error) {
	var scs storagev1.StorageClassList
	err := w.client.List(ctx, &scs, client.MatchingFields(map[string]string{resizeEnableIndexKey: "true"}))
	if err != nil {
		metrics.KubernetesClientFailTotal.Increment()
		return nil, err
	}
	return &scs, nil
}

func (w *pvcAutoresizer) reconcile(ctx context.Context) error {
	scs, err := w.getStorageClassList(ctx)
	if err != nil {
		w.log.Error(err, "getStorageClassList failed")
		return err
	}

	vsMap, err := w.metricsClient.GetMetrics(ctx)
	if err != nil {
		w.log.Error(err, "metricsClient.GetMetrics failed")
		return err
	}

	for _, sc := range scs.Items {
		var pvcs corev1.PersistentVolumeClaimList
		err = w.client.List(ctx, &pvcs, client.MatchingFields(map[string]string{storageClassNameIndexKey: sc.Name}))
		if err != nil {
			metrics.KubernetesClientFailTotal.Increment()
			w.log.Error(err, "list pvc failed")
			return err
		}
		for _, pvc := range pvcs.Items {
			log := w.log.WithValues("namespace", pvc.Namespace, "name", pvc.Name)
			isTarget, err := isTargetPVC(&pvc)
			if err != nil {
				metrics.ResizerFailedResizeTotal.Increment(pvc.Name, pvc.Namespace)
				log.Error(err, "failed to check target PVC")
				continue
			} else if !isTarget {
				continue
			}

			// To output the metric even if some events do not occur, we call SpecifyLabels() here.
			metrics.ResizerSuccessResizeTotal.SpecifyLabels(pvc.Name, pvc.Namespace)
			metrics.ResizerFailedResizeTotal.SpecifyLabels(pvc.Name, pvc.Namespace)
			metrics.ResizerLimitReachedTotal.SpecifyLabels(pvc.Name, pvc.Namespace)

			namespacedName := types.NamespacedName{
				Namespace: pvc.Namespace,
				Name:      pvc.Name,
			}
			if _, ok := vsMap[namespacedName]; !ok {
				log.Info("failed to get volume stats")
				continue
			}

			if err := w.resize(ctx, &pvc, vsMap[namespacedName]); err != nil {
				metrics.ResizerFailedResizeTotal.Increment(pvc.Name, pvc.Namespace)
				log.Error(err, "failed to resize PVC")
			}
		}
	}

	return nil
}

func (w *pvcAutoresizer) resize(ctx context.Context, pvc *corev1.PersistentVolumeClaim, vs *VolumeStats) error {
	log := w.log.WithName("resize").WithValues("namespace", pvc.Namespace, "name", pvc.Name)

	threshold, err := convertSizeInBytes(pvc.Annotations[pvcautoresizer.ResizeThresholdAnnotation], vs.CapacityBytes, pvcautoresizer.DefaultThreshold)
	if err != nil {
		log.V(logLevelWarn).Info("failed to convert threshold annotation", "error", err.Error())
		// lint:ignore nilerr ignores this because invalid annotations should be allowed.
		return nil
	}

	annotation := pvc.Annotations[pvcautoresizer.ResizeInodesThresholdAnnotation]
	inodesThreshold, err := convertSize(annotation, vs.CapacityInodeSize, pvcautoresizer.DefaultInodesThreshold)
	if err != nil {
		log.V(logLevelWarn).Info("failed to convert threshold annotation", "error", err.Error())
		// lint:ignore nilerr ignores this because invalid annotations should be allowed.
		return nil
	}

	cap, exists := pvc.Status.Capacity[corev1.ResourceStorage]
	if !exists {
		log.Info("skip resizing because pvc capacity is not set yet")
		return nil
	}
	if cap.Value() == 0 {
		log.Info("skip resizing because pvc capacity size is zero")
		return nil
	}

	increase, err := convertSizeInBytes(pvc.Annotations[pvcautoresizer.ResizeIncreaseAnnotation], cap.Value(), pvcautoresizer.DefaultIncrease)
	if err != nil {
		log.V(logLevelWarn).Info("failed to convert increase annotation", "error", err.Error())
		return nil
	}

	preCap, exist := pvc.Annotations[pvcautoresizer.PreviousCapacityBytesAnnotation]
	if exist {
		preCapInt64, err := strconv.ParseInt(preCap, 10, 64)
		if err != nil {
			log.V(logLevelWarn).Info("failed to parse pre_cap_bytes annotation", "error", err.Error())
			// lint:ignore nilerr ignores this because invalid annotations should be allowed.
			return nil
		}
		if preCapInt64 == vs.CapacityBytes {
			log.Info("waiting for resizing...", "capacity", vs.CapacityBytes)
			return nil
		}
	}
	limitRes, err := PvcStorageLimit(pvc)
	if err != nil {
		log.Error(err, "fetching storage limit failed")
		if w.slackNotifier != nil {
			_ = w.slackNotifier.SendResizeNotification(pvc.Namespace, pvc.Name, cap.Value(), cap.Value(), false)
		}
		return err
	}

	// Check if current size is approaching the limit
	if !limitRes.IsZero() {
		currentSizePercent := float64(cap.Value()) / float64(limitRes.Value()) * 100
		if currentSizePercent >= float64(limitWarningThresholdPercent) && currentSizePercent < 100 {
			// Check if we've already sent a warning recently
			shouldSendWarning := true
			if lastWarningStr, exists := pvc.Annotations[lastWarningTimestampAnnotation]; exists {
				lastWarning, err := strconv.ParseInt(lastWarningStr, 10, 64)
				if err == nil {
					// Only send warning if more than 1 hour has passed since the last warning
					if time.Since(time.Unix(lastWarning, 0)) < time.Hour {
						shouldSendWarning = false
					}
				}
			}

			if shouldSendWarning {
				log.Info("volume size approaching limit",
					"currentSize", cap.Value(),
					"limit", limitRes.Value(),
					"percentage", currentSizePercent)
				
				if w.slackNotifier != nil {
					message := fmt.Sprintf("PVC is approaching storage limit (%.1f%% of %s)", 
						currentSizePercent, limitRes.String())
					if err := w.slackNotifier.SendLimitWarningNotification(pvc.Namespace, pvc.Name, cap.Value(), limitRes.Value(), message); err != nil {
						log.Error(err, "failed to send limit warning notification")
					} else {
						// Update the annotation with current timestamp
						if pvc.Annotations == nil {
							pvc.Annotations = make(map[string]string)
						}
						pvc.Annotations[lastWarningTimestampAnnotation] = strconv.FormatInt(time.Now().Unix(), 10)
						if err := w.client.Update(ctx, pvc); err != nil {
							log.Error(err, "failed to update last warning timestamp annotation")
						}
					}
				}
			}
		}
	}

	if cap.Cmp(limitRes) >= 0 {
		// Check if we've already sent a limit reached notification recently
		shouldSendNotification := true
		if lastNotificationStr, exists := pvc.Annotations[lastLimitReachedTimestampAnnotation]; exists {
			lastNotification, err := strconv.ParseInt(lastNotificationStr, 10, 64)
			if err == nil {
				// Only send notification if more than 1 hour has passed since the last one
				if time.Since(time.Unix(lastNotification, 0)) < time.Hour {
					shouldSendNotification = false
				}
			}
		}

		currentSizeStr := resource.NewQuantity(cap.Value(), resource.BinarySI).String()
		log.Info("volume storage limit reached",
			"currentSize", currentSizeStr,
			"limit", limitRes.String())
		metrics.ResizerLimitReachedTotal.Increment(pvc.Name, pvc.Namespace)

		if shouldSendNotification && w.slackNotifier != nil {
			_ = w.slackNotifier.SendResizeNotification(pvc.Namespace, pvc.Name, cap.Value(), cap.Value(), false)
			
			// Update the annotation with current timestamp
			if pvc.Annotations == nil {
				pvc.Annotations = make(map[string]string)
			}
			pvc.Annotations[lastLimitReachedTimestampAnnotation] = strconv.FormatInt(time.Now().Unix(), 10)
			if err := w.client.Update(ctx, pvc); err != nil {
				log.Error(err, "failed to update last limit reached timestamp annotation")
			}
		}
		return nil
	}

	if threshold > vs.AvailableBytes || inodesThreshold > vs.AvailableInodeSize {
		if pvc.Annotations == nil {
			pvc.Annotations = make(map[string]string)
		}
		newReqBytes := int64(math.Ceil(float64(cap.Value()+increase)/(1<<30))) << 30
		newReq := resource.NewQuantity(newReqBytes, resource.BinarySI)
		if newReq.Cmp(limitRes) > 0 {
			newReq = &limitRes
		}

		oldSize := cap.Value()
		pvc.Spec.Resources.Requests[corev1.ResourceStorage] = *newReq
		pvc.Annotations[pvcautoresizer.PreviousCapacityBytesAnnotation] = strconv.FormatInt(vs.CapacityBytes, 10)
		err = w.client.Update(ctx, pvc)
		if err != nil {
			metrics.KubernetesClientFailTotal.Increment()
			if w.slackNotifier != nil {
				_ = w.slackNotifier.SendResizeNotification(pvc.Namespace, pvc.Name, oldSize, newReq.Value(), false)
			}
			return err
		}
		log.Info("resize started",
			"from", cap.Value(),
			"to", newReq.Value(),
			"threshold", threshold,
			"available", vs.AvailableBytes,
			"inodesThreshold", inodesThreshold,
			"inodesAvailable", vs.AvailableInodeSize,
		)
		w.recorder.Eventf(pvc, corev1.EventTypeNormal, "Resized", "PVC volume is resized to %s", newReq.String())
		metrics.ResizerSuccessResizeTotal.Increment(pvc.Name, pvc.Namespace)
		if w.slackNotifier != nil {
			_ = w.slackNotifier.SendResizeNotification(pvc.Namespace, pvc.Name, oldSize, newReq.Value(), true)
		}
	}

	return nil
}

func indexByResizeEnableAnnotation(obj client.Object) []string {
	sc := obj.(*storagev1.StorageClass)
	if val, ok := sc.Annotations[pvcautoresizer.AutoResizeEnabledKey]; ok {
		return []string{val}
	}

	return []string{}
}

func indexByStorageClassName(obj client.Object) []string {
	pvc := obj.(*corev1.PersistentVolumeClaim)
	scName := pvc.Spec.StorageClassName
	if scName == nil {
		return []string{}
	}
	return []string{*scName}
}

// SetupIndexer setup indices for PVC auto resizer
func SetupIndexer(mgr ctrl.Manager, skipAnnotationCheck bool) error {
	idxFunc := indexByResizeEnableAnnotation
	if skipAnnotationCheck {
		idxFunc = func(_ client.Object) []string { return []string{"true"} }
	}
	err := mgr.GetFieldIndexer().IndexField(context.Background(), &storagev1.StorageClass{}, resizeEnableIndexKey, idxFunc)
	if err != nil {
		return err
	}

	err = mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.PersistentVolumeClaim{}, storageClassNameIndexKey,
		indexByStorageClassName)
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
		return calcSize(valStr, capacity)
	}

	quantity, err := resource.ParseQuantity(valStr)
	if err != nil {
		return 0, err
	}
	val := quantity.Value()
	if val <= 0 {
		return 0, fmt.Errorf("annotation value should be positive: %s", valStr)
	}
	return val, nil
}

func convertSize(valStr string, capacity int64, defaultVal string) (int64, error) {
	if len(valStr) == 0 {
		valStr = defaultVal
	}
	if strings.HasSuffix(valStr, "%") {
		return calcSize(valStr, capacity)
	}
	return 0, fmt.Errorf("annotation value should be in percent notation: %s", valStr)
}

func calcSize(valStr string, capacity int64) (int64, error) {
	rate, err := strconv.ParseFloat(strings.TrimRight(valStr, "%"), 64)
	if err != nil {
		return 0, err
	}
	if rate < 0 || rate > 100 {
		return 0, fmt.Errorf("annotation value should between 0 and 100: %s", valStr)
	}

	res := int64(float64(capacity) * rate / 100.0)
	return res, nil
}

func PvcStorageLimit(pvc *corev1.PersistentVolumeClaim) (resource.Quantity, error) {
	// storage limit on the annotation has precedence
	if annotation, ok := pvc.Annotations[pvcautoresizer.StorageLimitAnnotation]; ok && annotation != "" {
		return resource.ParseQuantity(annotation)
	}

	return *resource.NewQuantity(0, resource.BinarySI), nil
}
