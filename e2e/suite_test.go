package e2e

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

var (
	binDir     string
	failedTest bool

	//go:embed testdata/pod-pvc-template.yaml
	podPVCTemplateYAML string

	podPVCTemplateOnce sync.Once
	podPVCTmpl         *template.Template

	testNamespace string = "autoresizer-test"

	consistentlyTimeout time.Duration = 20 * time.Second
)

func execAtLocal(cmd string, input []byte, args ...string) ([]byte, []byte, error) {
	var stdout, stderr bytes.Buffer
	command := exec.Command(cmd, args...)
	command.Stdout = &stdout
	command.Stderr = &stderr

	if len(input) != 0 {
		command.Stdin = bytes.NewReader(input)
	}

	err := command.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func kubectl(args ...string) ([]byte, []byte, error) {
	return execAtLocal(filepath.Join(binDir, "kubectl"), nil, args...)
}

func kubectlWithInput(input []byte, args ...string) ([]byte, []byte, error) {
	return execAtLocal(filepath.Join(binDir, "kubectl"), input, args...)
}

func TestMtest(t *testing.T) {
	if os.Getenv("E2ETEST") == "" {
		t.Skip("Run under e2e/")
		return
	}
	rand.Seed(time.Now().UnixNano())

	RegisterFailHandler(Fail)

	SetDefaultEventuallyPollingInterval(time.Second)
	SetDefaultEventuallyTimeout(5 * time.Minute)

	RunSpecs(t, "Test on sanity")
}

func createNamespace(ns string) {
	stdout, stderr, err := kubectl("create", "namespace", ns)
	Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
	Eventually(func() error {
		return waitCreatingDefaultSA(ns)
	}).Should(Succeed())
}

func waitCreatingDefaultSA(ns string) error {
	stdout, stderr, err := kubectl("get", "sa", "-n", ns, "default")
	if err != nil {
		return fmt.Errorf("default sa is not found. stdout=%s, stderr=%s, err=%v", stdout, stderr, err)
	}
	return nil
}

var _ = BeforeSuite(func() {
	By("[BeforeSuite] Getting the directory path which contains some binaries")
	binDir = os.Getenv("BINDIR")
	Expect(binDir).ShouldNot(BeEmpty())
	fmt.Println("This test uses the binaries under " + binDir)

	By("[BeforeSuite] Waiting for pvc-autoresizer-controller to get ready")
	Eventually(func() error {
		stdout, stderr, err := kubectl("-n", "pvc-autoresizer", "get", "deploy", "pvc-autoresizer-controller", "-o", "json")
		if err != nil {
			return errors.New(string(stderr))
		}

		var deploy appsv1.Deployment
		err = yaml.Unmarshal(stdout, &deploy)
		if err != nil {
			return err
		}

		if deploy.Status.AvailableReplicas != 1 {
			return errors.New("pvc-autoresizer-controller is not available yet")
		}

		return nil
	}).Should(Succeed())

	By("[BeforeSuite] Creating namespace for test")
	createNamespace(testNamespace)
})

var _ = AfterSuite(func() {
	if !failedTest {
		By("[AfterSuite] Delete namespace for autoresizer tests")
		stdout, stderr, err := kubectl("delete", "namespace", testNamespace)
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
	}
})

type resource struct {
	resource string
	name     string
}

var _ = Describe("pvc-autoresizer", func() {
	var resources []resource

	var _ = AfterEach(func() {
		if CurrentSpecReport().Failed() {
			failedTest = true
		} else {
			By("[AfterEach] cleanup resources")
			for _, r := range resources {
				stdout, stderr, err := kubectl("-n", testNamespace, "delete", r.resource, r.name)
				Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
			}
		}
		resources = []resource{}
	})

	It("should resize PVC based on PVC spec and annotations", func() {
		pvcName := "test-pvc"
		sc := "topolvm-provisioner-annotated"
		mode := string(corev1.PersistentVolumeFilesystem)
		request := "1Gi"
		limit := "2Gi"
		threshold := "50%"
		increase := "1Gi"
		storageLimit := ""

		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request, limit, threshold, "", increase, storageLimit)

		By("create a file with a size that does not exceed threshold disk usage")
		stdout, stderr, err := kubectl("-n", testNamespace, "exec", "-it", pvcName, "--", "fallocate", "-l", "400M", "/test1/test1.txt")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk does not resize")
		checkDoesNotResize(pvcName, request)

		By("add a file with a size that exceed threshold disk usage")
		stdout, stderr, err = kubectl("-n", testNamespace, "exec", "-it", pvcName, "--", "fallocate", "-l", "200M", "/test1/test2.txt")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk resizing")
		checkDiskResize(pvcName, "2Gi", true)

		By("add a file with a size that exceed threshold disk usage too")
		stdout, stderr, err = kubectl("-n", testNamespace, "exec", "-it", pvcName, "--", "fallocate", "-l", "1G", "/test1/test3.txt")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk does not resize when PVC resource limit has come")
		checkDoesNotResize(pvcName, "2Gi")
	})

	It("should resize PVC based on pvc-autoresizer default setting", func() {
		pvcName := "test-pvc-nonannotation"
		sc := "topolvm-provisioner-annotated"
		mode := string(corev1.PersistentVolumeFilesystem)
		request := "10Gi"
		limit := "11Gi"
		threshold := ""
		increase := ""
		storageLimit := ""

		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request, limit, threshold, "", increase, storageLimit)

		By("create a file with a size that does not exceed threshold disk usage")
		stdout, stderr, err := kubectl("-n", testNamespace, "exec", "-it", pvcName, "--", "fallocate", "-l", "9G", "/test1/test1.txt")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk resizing")
		checkDiskResize(pvcName, "11Gi", false)
	})

	It("should resize PVC based on PVC spec and annotations(with storage limit annotation)", func() {
		pvcName := "test-pvc-storage-limit"
		sc := "topolvm-provisioner-annotated"
		mode := string(corev1.PersistentVolumeFilesystem)
		request := "1Gi"
		limit := "3Gi"
		threshold := "50%"
		increase := "1Gi"
		storageLimit := "2Gi"

		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request, limit, threshold, "", increase, storageLimit)

		By("create a file with a size that exceed threshold disk usage")
		stdout, stderr, err := kubectl("-n", testNamespace, "exec", "-it", pvcName, "--", "fallocate", "-l", "600M", "/test1/test1.txt")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk resizing")
		checkDiskResize(pvcName, "2Gi", true)

		By("add a file with a size that exceed threshold disk usage too")
		stdout, stderr, err = kubectl("-n", testNamespace, "exec", "-it", pvcName, "--", "fallocate", "-l", "1G", "/test1/test2.txt")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk does not resize when PVC storage limit annotation limit has come")
		checkDoesNotResize(pvcName, "2Gi")
	})

	It("should not resize PVC with Block mode", func() {
		pvcName := "test-pvc-block-mode"
		sc := "topolvm-provisioner-annotated"
		mode := string(corev1.PersistentVolumeBlock)
		request := "1Gi"
		limit := "2Gi"
		threshold := "50%"
		increase := "1Gi"
		storageLimit := ""

		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request, limit, threshold, "", increase, storageLimit)

		By("write data with a size that exceed threshold disk usage")
		stdout, stderr, err := kubectl("-n", testNamespace, "exec", "-it", pvcName, "--", "dd", "if=/dev/zero", "of=/dev/e2etest", "count=600M", "iflag=count_bytes")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk does not resize")
		checkDoesNotResize(pvcName, "1Gi")
	})

	It("should not resize PVC without available StorageClass", func() {
		pvcName := "test-pvc-with-disable-pvc-autoresizer"
		sc := "topolvm-provisioner"
		mode := string(corev1.PersistentVolumeFilesystem)
		request := "1Gi"
		limit := "2Gi"
		threshold := "50%"
		increase := "1Gi"
		storageLimit := ""

		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request, limit, threshold, "", increase, storageLimit)

		By("create a file with a size that exceed threshold disk usage")
		stdout, stderr, err := kubectl("-n", testNamespace, "exec", "-it", pvcName, "--", "fallocate", "-l", "600M", "/test1/test1.txt")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk does not resize")
		checkDoesNotResize(pvcName, "1Gi")
	})

	It("should resize PVC based on PVC spec and annotations(inode)", func() {
		pvcName := "test-pvc"
		sc := "topolvm-provisioner-annotated"
		mode := string(corev1.PersistentVolumeFilesystem)
		request := "1Gi"
		limit := "2Gi"
		threshold := "50%"
		increase := "1Gi"
		inodesThreshold := "99%"
		storageLimit := ""

		var capacityInode int64
		var availableInode int64

		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request, limit, threshold, inodesThreshold, increase, storageLimit)

		By("getting available inode size and capacity inode size")
		stdout, stderr, err := kubectl("-n", testNamespace, "exec", "-it", pvcName, "--", "df", "/test1", "--output=target,itotal,iavail")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		lines := regexp.MustCompile(`\n`).Split(string(stdout), -1)
		Expect(len(lines)).Should(Equal(3))

		cs := regexp.MustCompile(`\s+`).Split(string(lines[1]), -1)
		capacityInode, err = strconv.ParseInt(cs[1], 10, 64)
		Expect(err).ShouldNot(HaveOccurred())
		availableInode, err = strconv.ParseInt(cs[2], 10, 64)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(capacityInode).ShouldNot(Equal(int64(0)))
		Expect(availableInode).ShouldNot(Equal(int64(0)))

		By("checking the disk does not resize")
		checkDoesNotResize(pvcName, request)

		By("create files for consume an inodes")
		rate, err := strconv.ParseFloat(strings.TrimRight(inodesThreshold, "%"), 64)
		Expect(err).ShouldNot(HaveOccurred())
		th := int64(float64(capacityInode) * rate / 100.0)
		num := capacityInode - th
		stdout, stderr, err = kubectl("-n", testNamespace, "exec", "-it", pvcName, "--", "bash", "-c", fmt.Sprintf("touch /test1/testfile_{0..%d}.txt", num))
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk resizing")
		checkDiskResize(pvcName, "2Gi", true)
	})
})

func buildPodPVCTemplateYAML(pvcName, storageClassName, volumeMode, podName, request, limit, threshold, inodesThreshold, increase, storageLimit string) ([]byte, error) {
	var b bytes.Buffer
	var err error

	useAnnotation := "false"
	if threshold != "" || increase != "" || storageLimit != "" {
		useAnnotation = "true"
	}

	podPVCTemplateOnce.Do(func() {
		podPVCTmpl, err = template.New("").Parse(podPVCTemplateYAML)
	})
	if err != nil {
		return b.Bytes(), err
	}

	params := map[string]string{
		"pvcName":                   pvcName,
		"storageClassName":          storageClassName,
		"volumeMode":                volumeMode,
		"podName":                   podName,
		"namespace":                 testNamespace,
		"useAnnotation":             useAnnotation,
		"thresholdAnnotation":       threshold,
		"increaseAnnotation":        increase,
		"inodesThresholdAnnotation": inodesThreshold,
		"storageLimitAnnotation":    storageLimit,
		"resourceRequest":           request,
		"resourceLimit":             limit,
	}
	err = podPVCTmpl.Execute(&b, params)
	return b.Bytes(), err
}

func createPodPVC(resources []resource, pvcName, storageClassName, volumeMode, podName, request, limit, threshold, inodesThreshold, increase, storageLimit string) []resource {
	By("create a PVC and a pod for test")
	podPVCYAML, err := buildPodPVCTemplateYAML(pvcName, storageClassName, volumeMode, pvcName, request, limit, threshold, inodesThreshold, increase, storageLimit)
	Expect(err).ShouldNot(HaveOccurred())
	stdout, stderr, err := kubectlWithInput(podPVCYAML, "apply", "-f", "-")
	Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s yaml=\n%s", stdout, stderr, podPVCYAML)
	resources = append(resources, resource{resource: "pod", name: pvcName})
	resources = append(resources, resource{resource: "pvc", name: pvcName})

	By("waiting for creating the volume and running the pod")
	Eventually(func() error {
		stdout, stderr, err := kubectl("get", "-n", testNamespace, "pod", pvcName, "-o", "yaml")
		if err != nil {
			return fmt.Errorf("failed to get pod name of %s/%s. stdout: %s, stderr: %s, err: %v", testNamespace, pvcName, stdout, stderr, err)
		}

		var pod corev1.Pod
		err = yaml.Unmarshal(stdout, &pod)
		if err != nil {
			return err
		}

		if pod.Status.Phase != corev1.PodRunning {
			return errors.New("Pod is not running")
		}

		return nil
	}).Should(Succeed())

	return resources
}

func checkDoesNotResize(pvcName, expect string) {
	Consistently(func() error {
		stdout, stderr, err := kubectl("-n", testNamespace, "get", "pvc", pvcName, "-o", "json")
		if err != nil {
			return fmt.Errorf("failed to get PVC. stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		}

		var pvc corev1.PersistentVolumeClaim
		err = json.Unmarshal(stdout, &pvc)
		if err != nil {
			return fmt.Errorf("failed to unmarshal PVC. stdout: %s, err: %v", stdout, err)
		}

		actual := pvc.Spec.Resources.Requests.Storage().String()
		if actual != expect {
			return fmt.Errorf("pvc resource request is not %q: actual=%q", expect, actual)
		}

		return nil
	}, consistentlyTimeout).ShouldNot(HaveOccurred())
}

func checkDiskResize(pvcName, expect string, checkCapacity bool) {
	Eventually(func() error {
		stdout, stderr, err := kubectl("-n", testNamespace, "get", "pvc", pvcName, "-o", "json")
		if err != nil {
			return fmt.Errorf("failed to get PVC. stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		}

		var pvc corev1.PersistentVolumeClaim
		err = json.Unmarshal(stdout, &pvc)
		if err != nil {
			return fmt.Errorf("failed to unmarshal PVC. stdout: %s, err: %v", stdout, err)
		}

		actual := pvc.Spec.Resources.Requests.Storage().String()
		if actual != expect {
			return fmt.Errorf("pvc resource request is not %q: actual=%q", expect, actual)
		}

		if checkCapacity {
			actual = pvc.Status.Capacity.Storage().String()
			if actual != expect {
				return fmt.Errorf("pvc capacity is not %q: actual=%q", expect, actual)
			}
		}

		return nil
	}).ShouldNot(HaveOccurred())
}
