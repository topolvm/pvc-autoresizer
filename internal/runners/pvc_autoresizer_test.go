package runners

import (
	"context"
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pvcautoresizer "github.com/topolvm/pvc-autoresizer"
	corev1 "k8s.io/api/core/v1"
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
				valStr:     "101%",
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

	Context("test convertSize", func() {
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
		}
		errorCases := []input{
			{
				valStr:     "10",
				capacity:   100,
				defaultVal: "10%",
			},
			{
				valStr:     "-10%",
				capacity:   100,
				defaultVal: "10%",
			},
			{
				valStr:     "101%",
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
				res, err := convertSize(val.input.valStr, val.input.capacity, val.input.defaultVal)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(val.expect))
			}
		})
		It("should be error", func() {
			for _, val := range errorCases {
				_, err := convertSize(val.valStr, val.capacity, val.defaultVal)
				Expect(err).To(HaveOccurred(), "%+v", val)
			}
		})
	})

	Context("resize", func() {
		Context("parameter tests", func() {
			ctx := context.Background()
			pvcNS := "default"
			increase := "1Gi"
			limit := int64(100 << 30)
			volumeMode := corev1.PersistentVolumeFilesystem

			testCases := []struct {
				description        string
				pvcSizeGi          int64
				pvcCapSizeGi       int64
				expectSizeGi       int64
				threshold          string
				availableByte      int64
				capacityInodeSize  int64
				availableInodeSize int64
				inodesThreshold    string
			}{
				{
					description:        "Should resize(absolute value)",
					pvcSizeGi:          10,
					pvcCapSizeGi:       10,
					expectSizeGi:       11,
					threshold:          "5Gi",
					availableByte:      5<<30 - 1,
					availableInodeSize: 100,
					capacityInodeSize:  100,
				},
				{
					description:        "Should not resize(absolute value)",
					pvcSizeGi:          10,
					pvcCapSizeGi:       10,
					expectSizeGi:       10,
					threshold:          "5Gi",
					availableByte:      5 << 30,
					availableInodeSize: 100,
					capacityInodeSize:  100,
				},
				{
					description:        "Should resize(%)",
					pvcSizeGi:          10,
					pvcCapSizeGi:       10,
					expectSizeGi:       11,
					threshold:          "50%",
					availableByte:      5<<30 - 1,
					availableInodeSize: 100,
					capacityInodeSize:  100,
				},
				{
					description:        "Should not resize(%)",
					pvcSizeGi:          10,
					pvcCapSizeGi:       10,
					expectSizeGi:       10,
					threshold:          "50%",
					availableByte:      5 << 30,
					availableInodeSize: 100,
					capacityInodeSize:  100,
				},
				{
					description:        "Should resize(inode)",
					pvcSizeGi:          10,
					pvcCapSizeGi:       10,
					expectSizeGi:       11,
					threshold:          "50%",
					availableByte:      5 << 30,
					availableInodeSize: 9,
					capacityInodeSize:  100,
				},
				{
					description:        "Should resize(inode with annotation)",
					pvcSizeGi:          10,
					pvcCapSizeGi:       10,
					expectSizeGi:       11,
					threshold:          "50%",
					availableByte:      5 << 30,
					availableInodeSize: 49,
					capacityInodeSize:  100,
					inodesThreshold:    "50%",
				},
				{
					description:        "Should not resize(inode)",
					pvcSizeGi:          10,
					pvcCapSizeGi:       10,
					expectSizeGi:       10,
					threshold:          "50%",
					availableByte:      5 << 30,
					availableInodeSize: 9,
					capacityInodeSize:  100,
					inodesThreshold:    "0%",
				},
				{
					description:        "Should resize(capacity size check)",
					pvcSizeGi:          1,
					pvcCapSizeGi:       10,
					expectSizeGi:       11,
					threshold:          "5Gi",
					availableByte:      5<<30 - 1,
					availableInodeSize: 100,
					capacityInodeSize:  100,
				},
				{
					description:        "Should not resize(inode - 0 capacityInodeSize)",
					pvcSizeGi:          10,
					pvcCapSizeGi:       10,
					expectSizeGi:       10,
					threshold:          "50%",
					availableByte:      5 << 30,
					availableInodeSize: 0,
					capacityInodeSize:  0,
					inodesThreshold:    "20%",
				},
				{
					description:        "Should not resize(no capacity value set)",
					pvcSizeGi:          10,
					pvcCapSizeGi:       -1,
					expectSizeGi:       10,
					threshold:          "5Gi",
					availableByte:      5<<30 - 1,
					availableInodeSize: 100,
					capacityInodeSize:  100,
				},
			}

			for i, tc := range testCases {
				pvcName := fmt.Sprintf("test-pvc-%d", i)
				pvcSizeGi := tc.pvcSizeGi
				pvcCapSizeGi := tc.pvcCapSizeGi
				expectSizeGi := tc.expectSizeGi
				threshold := tc.threshold
				availableByte := tc.availableByte
				availableInodeSize := tc.availableInodeSize
				capacityInodeSize := tc.capacityInodeSize
				inodesThreshold := tc.inodesThreshold

				description := fmt.Sprintf(
					"%s: pvcSizeGi=%d expectSizeGi=%d threshold=%q availableByte=%d availableInodeSize=%d "+
						"capacityInodeSize=%d inodesThreshold=%q",
					tc.description,
					tc.pvcSizeGi,
					tc.expectSizeGi,
					tc.threshold,
					tc.availableByte,
					availableInodeSize,
					capacityInodeSize,
					inodesThreshold)

				It(description, func() {
					createPVC(ctx, pvcNS, pvcName, scName, threshold, inodesThreshold, increase, pvcSizeGi<<30, limit,
						pvcCapSizeGi<<30, volumeMode)
					setMetrics(pvcNS, pvcName, availableByte, pvcSizeGi<<30, availableInodeSize, capacityInodeSize)
					testFunc := func() error {
						var pvc corev1.PersistentVolumeClaim
						err := k8sClient.Get(ctx, types.NamespacedName{Namespace: pvcNS, Name: pvcName}, &pvc)
						if err != nil {
							return err
						}
						req := pvc.Spec.Resources.Requests.Storage().Value()

						ALLOWANCE := int64(1 << 10)
						if !(expectSizeGi<<30-ALLOWANCE < req && req <= expectSizeGi<<30+ALLOWANCE) {
							return fmt.Errorf("request size(Gi) should be %d, but %d", expectSizeGi, req>>30)
						}
						return nil
					}

					if pvcSizeGi == expectSizeGi {
						Consistently(testFunc, 3*time.Second).ShouldNot(HaveOccurred())
					} else {
						Eventually(testFunc, 3*time.Second).ShouldNot(HaveOccurred())
					}
				})
			}
		})

		Context("metrics tests", func() {
			It("should output metrics", func() {
				ctx := context.Background()
				pvcNS := "default"
				pvcName := "test-resize-metrics"
				createPVC(ctx, pvcNS, pvcName, scName, "50%", "", "20Gi", 10<<30, 100<<30, 10<<30,
					corev1.PersistentVolumeFilesystem)
				By("running resize", func() {
					setMetrics(pvcNS, pvcName, 3<<30, 7<<30, 2050246, 2050246)
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

				By("checking metrics", func() {
					mfs, err := getMetricsFamily()
					Expect(err).NotTo(HaveOccurred())
					mf, ok := mfs["pvcautoresizer_loop_seconds_total"]
					Expect(ok).To(BeTrue())

					var val float64
					for _, m := range mf.Metric {
						if m.Counter == nil {
							continue
						}
						if m.Counter.Value == nil {
							continue
						}
						val = *m.Counter.Value
					}
					Expect(val).NotTo(Equal(float64(0)))

					mf, ok = mfs["pvcautoresizer_success_resize_total"]
					Expect(ok).To(BeTrue())
					var val2 int
					for _, m := range mf.Metric {
						if m.Counter == nil {
							continue
						}
						if m.Counter.Value == nil {
							continue
						}
						val2 = int(*m.Counter.Value)
					}
					Expect(val2).NotTo(Equal(0))

					// This metrics output from the pvcAutoresizer with FakeClientWrapper
					mf, ok = mfs["pvcautoresizer_failed_resize_total"]
					Expect(ok).To(BeTrue())
					var val3 int
					for _, m := range mf.Metric {
						if m.Counter == nil {
							continue
						}
						if m.Counter.Value == nil {
							continue
						}
						val3 = int(*m.Counter.Value)
					}
					Expect(val3).NotTo(Equal(0))

					// This metrics output from the pvcAutoresizer with FakeClientWrapper
					mf, ok = mfs["pvcautoresizer_kubernetes_client_fail_total"]
					Expect(ok).To(BeTrue())
					var val4 int
					for _, m := range mf.Metric {
						if m.Counter == nil {
							continue
						}
						if m.Counter.Value == nil {
							continue
						}
						val4 = int(*m.Counter.Value)
					}
					Expect(val4).NotTo(Equal(0))
				})
			})
		})
	})
})

func createPVC(ctx context.Context, ns, name, scName, threshold, inodesThreshold, increase string,
	request, limit, capacity int64, mode corev1.PersistentVolumeMode,
) {
	pvc := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   ns,
			Annotations: map[string]string{},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: *resource.NewQuantity(request, resource.BinarySI),
				},
			},
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &scName,
			VolumeMode:       &mode,
		},
	}

	if len(threshold) != 0 {
		pvc.Annotations[pvcautoresizer.ResizeThresholdAnnotation] = threshold
	}
	if len(inodesThreshold) != 0 {
		pvc.Annotations[pvcautoresizer.ResizeInodesThresholdAnnotation] = inodesThreshold
	}

	if len(increase) != 0 {
		pvc.Annotations[pvcautoresizer.ResizeIncreaseAnnotation] = increase
	}

	if limit != 0 {
		pvc.Annotations[pvcautoresizer.StorageLimitAnnotation] = strconv.FormatInt(limit, 10)
	}

	err := k8sClient.Create(ctx, &pvc)
	Expect(err).NotTo(HaveOccurred())

	pvc.Status.Phase = corev1.ClaimBound
	if capacity >= 0 {
		pvc.Status.Capacity = map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceStorage: *resource.NewQuantity(capacity, resource.BinarySI),
		}
	}
	err = k8sClient.Status().Update(ctx, &pvc)
	Expect(err).NotTo(HaveOccurred())
}

func setMetrics(ns, name string, availableBytes, capacityBytes, availableInodeSize, capacityInodeSize int64) {
	promClient.setResponce(types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}, &VolumeStats{
		AvailableBytes:     availableBytes,
		CapacityBytes:      capacityBytes,
		AvailableInodeSize: availableInodeSize,
		CapacityInodeSize:  capacityInodeSize,
	})
}
