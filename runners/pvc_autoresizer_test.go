package runners

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
		Context("parameter tests", func() {
			ctx := context.Background()
			pvcNS := "default"
			increase := "1Gi"
			limit := int64(100 << 30)
			volumeMode := corev1.PersistentVolumeFilesystem

			testCases := []struct {
				description   string
				pvcSizeGi     int64
				expectSizeGi  int64
				threshold     string
				availableByte int64
			}{
				{
					description:   "Should resize(absolute value)",
					pvcSizeGi:     10,
					expectSizeGi:  11,
					threshold:     "5Gi",
					availableByte: 5<<30 - 1,
				},
				{
					description:   "Should not resize(absolute value)",
					pvcSizeGi:     10,
					expectSizeGi:  10,
					threshold:     "5Gi",
					availableByte: 5 << 30,
				},
				{
					description:   "Should resize(%)",
					pvcSizeGi:     10,
					expectSizeGi:  11,
					threshold:     "50%",
					availableByte: 5<<30 - 1,
				},
				{
					description:   "Should not resize(%)",
					pvcSizeGi:     10,
					expectSizeGi:  10,
					threshold:     "50%",
					availableByte: 5 << 30,
				},
			}

			for i, tc := range testCases {
				pvcName := fmt.Sprintf("test-pvc-%d", i)
				pvcSizeGi := tc.pvcSizeGi
				expectSizeGi := tc.expectSizeGi
				threshold := tc.threshold
				availableByte := tc.availableByte

				description := fmt.Sprintf(
					"%s: pvcSizeGi=%d expectSizeGi=%d threshold=%s availableByte=%d",
					tc.description,
					tc.pvcSizeGi,
					tc.expectSizeGi,
					tc.threshold,
					tc.availableByte)

				It(description, func() {
					createPVC(ctx, pvcNS, pvcName, scName, threshold, increase, pvcSizeGi<<30, limit, volumeMode)
					setMetrics(pvcNS, pvcName, availableByte, pvcSizeGi<<30)
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
	})
})

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

func setMetrics(ns, name string, available, capacity int64) {
	promClient.setResponce(types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}, &VolumeStats{
		AvailableBytes: available,
		CapacityBytes:  capacity,
	})
}
