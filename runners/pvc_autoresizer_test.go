package runners

import (
	"context"
	"fmt"
	"os"
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
				Expect(err).To(HaveOccurred(), "%+v", val)
			}
		})
	})

	Context("resize", func() {
		ctx := context.Background()
		provName := "test-provisioner"
		pvcNS := "default"

		It("should resize PVC", func() {
			scName := "test-storageclass1"
			pvcName := "tset-pvc1"
			createStorageClass(ctx, scName, provName, true)
			createPVC(ctx, pvcNS, pvcName, scName, "50%", "20Gi", 10<<30, 100<<30, corev1.PersistentVolumeFilesystem)

			By("60% available", func() {
				setMetrics(pvcNS, pvcName, 6<<30, 4<<30, 10<<30)
				Consistently(func() error {
					var pvc corev1.PersistentVolumeClaim
					err := k8sClient.Get(ctx, types.NamespacedName{Namespace: pvcNS, Name: pvcName}, &pvc)
					if err != nil {
						return err
					}
					req := pvc.Spec.Resources.Requests.Storage().Value()
					if req != 10<<30 {
						return fmt.Errorf("request size should be %d, but %d", 10<<30, req)
					}
					return nil
				}, 3*time.Second).ShouldNot(HaveOccurred())
			})

			By("30% available", func() {
				setMetrics(pvcNS, pvcName, 3<<30, 7<<30, 10<<30)
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
				}, 3*time.Second).ShouldNot(HaveOccurred())
			})
		})

		It("should resize PVC by default settings", func() {
			scName := "test-storageclass2"
			pvcName := "tset-pvc2"
			createStorageClass(ctx, scName, provName, true)
			createPVC(ctx, pvcNS, pvcName, scName, "", "", 10<<30, 100<<30, corev1.PersistentVolumeFilesystem)

			By("20% available", func() {
				setMetrics(pvcNS, pvcName, 2<<30, 8<<30, 10<<30)
				Consistently(func() error {
					var pvc corev1.PersistentVolumeClaim
					err := k8sClient.Get(ctx, types.NamespacedName{Namespace: pvcNS, Name: pvcName}, &pvc)
					if err != nil {
						return err
					}
					req := pvc.Spec.Resources.Requests.Storage().Value()
					if req != 10<<30 {
						return fmt.Errorf("request size should be %d, but %d", 10<<30, req)
					}
					return nil
				}, 3*time.Second).ShouldNot(HaveOccurred())
			})

			By("10% available", func() {
				setMetrics(pvcNS, pvcName, (1<<30)-1, (9<<30)+1, 10<<30)
				Eventually(func() error {
					var pvc corev1.PersistentVolumeClaim
					err := k8sClient.Get(ctx, types.NamespacedName{Namespace: pvcNS, Name: pvcName}, &pvc)
					if err != nil {
						return err
					}
					req := pvc.Spec.Resources.Requests.Storage().Value()
					if req != 11811160064 {
						return fmt.Errorf("request size should be %d, but %d", 11811160064, req)
					}
					return nil
				}, 3*time.Second).ShouldNot(HaveOccurred())
			})
		})

		It("should not resize PVC without limit", func() {
			scName := "test-storageclass3"
			pvcName := "tset-pvc3"
			createStorageClass(ctx, scName, provName, true)
			createPVC(ctx, pvcNS, pvcName, scName, "20%", "10Gi", 10<<30, 0, corev1.PersistentVolumeFilesystem)

			By("10% available", func() {
				setMetrics(pvcNS, pvcName, 1<<30, 9<<30, 10<<30)
				Consistently(func() error {
					var pvc corev1.PersistentVolumeClaim
					err := k8sClient.Get(ctx, types.NamespacedName{Namespace: pvcNS, Name: pvcName}, &pvc)
					if err != nil {
						return err
					}
					req := pvc.Spec.Resources.Requests.Storage().Value()
					if req != 10<<30 {
						return fmt.Errorf("request size should be %d, but %d", 10<<30, req)
					}
					return nil
				}, 3*time.Second).ShouldNot(HaveOccurred())
			})
		})

		It("should not resize PVC without available StorageClass", func() {
			scName := "test-storageclass4"
			pvcName := "tset-pvc4"
			createStorageClass(ctx, scName, provName, false)
			createPVC(ctx, pvcNS, pvcName, scName, "20%", "10Gi", 10<<30, 100<<30, corev1.PersistentVolumeFilesystem)

			By("10% available", func() {
				setMetrics(pvcNS, pvcName, 1<<30, 9<<30, 10<<30)
				noCheck := os.Getenv("NO_ANNOTATION_CHECK") == "true"
				if noCheck {
					Eventually(func() bool {
						var pvc corev1.PersistentVolumeClaim
						err := k8sClient.Get(ctx, types.NamespacedName{Namespace: pvcNS, Name: pvcName}, &pvc)
						if err != nil {
							return false
						}
						req := pvc.Spec.Resources.Requests.Storage().Value()
						return req > 10<<30
					}, 3*time.Second).Should(BeTrue())
				} else {
					Consistently(func() error {
						var pvc corev1.PersistentVolumeClaim
						err := k8sClient.Get(ctx, types.NamespacedName{Namespace: pvcNS, Name: pvcName}, &pvc)
						if err != nil {
							return err
						}
						req := pvc.Spec.Resources.Requests.Storage().Value()
						if req != 10<<30 {
							return fmt.Errorf("request size should be %d, but %d", 10<<30, req)
						}
						return nil
					}, 3*time.Second).ShouldNot(HaveOccurred())
				}
			})
		})

		It("should not resize PVC with Block mode", func() {
			scName := "test-storageclass5"
			pvcName := "tset-pvc5"
			createStorageClass(ctx, scName, provName, true)
			createPVC(ctx, pvcNS, pvcName, scName, "20%", "10Gi", 10<<30, 100<<30, corev1.PersistentVolumeBlock)

			By("10% available", func() {
				setMetrics(pvcNS, pvcName, 1<<30, 9<<30, 10<<30)
				Consistently(func() error {
					var pvc corev1.PersistentVolumeClaim
					err := k8sClient.Get(ctx, types.NamespacedName{Namespace: pvcNS, Name: pvcName}, &pvc)
					if err != nil {
						return err
					}
					req := pvc.Spec.Resources.Requests.Storage().Value()
					if req != 10<<30 {
						return fmt.Errorf("request size should be %d, but %d", 10<<30, req)
					}
					return nil
				}, 3*time.Second).ShouldNot(HaveOccurred())
			})
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

func createPVC(ctx context.Context, ns, name, scName, threshold, increase string, request, limit int64, mode corev1.PersistentVolumeMode) {
	pvc := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   ns,
			Annotations: map[string]string{},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: *resource.NewQuantity(request, resource.BinarySI),
				},
			},
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &scName,
			VolumeMode:       &mode,
		},
	}

	if len(threshold) != 0 {
		pvc.Annotations[ResizeThresholdAnnotation] = threshold
	}
	if len(increase) != 0 {
		pvc.Annotations[ResizeIncreaseAnnotation] = increase
	}

	if limit != 0 {
		pvc.Spec.Resources.Limits = map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceStorage: *resource.NewQuantity(limit, resource.BinarySI),
		}
	}

	err := k8sClient.Create(ctx, &pvc)
	Expect(err).NotTo(HaveOccurred())

	pvc.Status.Phase = corev1.ClaimBound
	err = k8sClient.Status().Update(ctx, &pvc)
	Expect(err).NotTo(HaveOccurred())
}

func setMetrics(ns, name string, available, used, capacity int64) {
	promClient.setResponce(types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}, &VolumeStats{
		AvailableBytes: available,
		UsedBytes:      used,
		CapacityBytes:  capacity,
	})
}
