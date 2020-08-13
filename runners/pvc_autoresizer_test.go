package runners

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("test resizer", func() {
	Context("test convertSizeInBytes", func() {
		type input struct {
			valStr     string
			capacity   int64
			defaultVal string
		}
		type testCase struct {
			input  input
			expect int64
		}
		correctCases := []testCase{
			{
				input: input{
					valStr:     "",
					capacity:   100,
					defaultVal: "10%",
				},
				expect: 10,
			},
			{
				input: input{
					valStr:     "20%",
					capacity:   100,
					defaultVal: "10%",
				},
				expect: 20,
			},
			{
				input: input{
					valStr:     "30Gi",
					capacity:   40 << 30,
					defaultVal: "10%",
				},
				expect: 30 << 30,
			},
			{
				input: input{
					valStr:     "100%",
					capacity:   100,
					defaultVal: "10%",
				},
				expect: 100,
			},
		}
		errorCases := []input{
			{
				valStr:     "-10%",
				capacity:   100,
				defaultVal: "10%",
			},
			{
				valStr:     "-10Gi",
				capacity:   100,
				defaultVal: "10%",
			},
			{
				valStr:     "101%",
				capacity:   100,
				defaultVal: "10%",
			},
			{
				valStr:     "11Gi",
				capacity:   10 << 30,
				defaultVal: "10%",
			},
			{
				valStr:     "hoge",
				capacity:   100,
				defaultVal: "10%",
			},
		}
		It("should be ok", func() {
			for _, val := range correctCases {
				res, err := convertSizeInBytes(val.input.valStr, val.input.capacity, val.input.defaultVal)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(val.expect))
			}
		})
		It("should be error", func() {
			for _, val := range errorCases {
				_, err := convertSizeInBytes(val.valStr, val.capacity, val.defaultVal)
				Expect(err).To(HaveOccurred())
			}
		})
	})

	Context("resize", func() {
		ctx := context.Background()
		scName := "test-storageclass"
		provName := "test-provisioner"
		pvcNS := "default"
		pvcName := "tset-pvc1"

		It("should resize PVC", func() {
			createStorageClass(ctx, scName, provName, true)
			createPVC(ctx, pvcNS, pvcName, scName, "50%", "20Gi", 10<<30, 100<<30)
			setMetrics(pvcNS, pvcName, 2<<30, 8<<30, 10<<30)

			Eventually(func() error {
				var pvc corev1.PersistentVolumeClaim
				err := k8sClient.Get(ctx, types.NamespacedName{Namespace: pvcNS, Name: pvcName}, &pvc)
				if err != nil {
					return err
				}
				req := pvc.Spec.Resources.Requests.Storage().Value()
				if req != 30<<30 {
					return fmt.Errorf("request size should be %d, but %d", 30<<30, req)
				}
				return nil
			}, 10*time.Second).ShouldNot(HaveOccurred())
		})
	})
})

func createStorageClass(ctx context.Context, name, provisioner string, enabled bool) {
	t := true
	sc := storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Provisioner:          provisioner,
		AllowVolumeExpansion: &t,
	}
	if enabled {
		sc.Annotations = map[string]string{
			AutoResizeEnabledKey: "true",
		}
	}
	err := k8sClient.Create(ctx, &sc)
	Expect(err).NotTo(HaveOccurred())
}

func createPVC(ctx context.Context, ns, name, scName, threshold, increase string, request, limit int64) {
	pvc := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Annotations: map[string]string{
				ResizeThresholdAnnotation: threshold,
				ResizeIncreaseAnnotation:  increase,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
				Limits: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: *resource.NewQuantity(limit, resource.BinarySI),
				},
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: *resource.NewQuantity(request, resource.BinarySI),
				},
			},
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &scName,
		},
	}
	err := k8sClient.Create(ctx, &pvc)
	Expect(err).NotTo(HaveOccurred())

	pvc.Status.Phase = corev1.ClaimBound
	err = k8sClient.Status().Update(ctx, &pvc)
	Expect(err).NotTo(HaveOccurred())
}

func setMetrics(ns, name string, available, used, capacity int64) {
	promClient.addResponse(types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}, &VolumeStats{
		AvailableBytes: available,
		UsedBytes:      used,
		CapacityBytes:  capacity,
	})
}
