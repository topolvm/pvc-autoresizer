package e2e

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
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

	testNamespace  string = "autoresizer-test"
	testNamespace2 string = "autoresizer-test2"

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
		t.Skip("Run under test/e2e/")
		return
	}

	RegisterFailHandler(Fail)

	SetDefaultEventuallyPollingInterval(time.Second)
	SetDefaultEventuallyTimeout(5 * time.Minute)
	EnforceDefaultTimeoutsWhenUsingContexts()

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

	By("[BeforeSuite] Waiting for mutating webhook working")
	podPVCYAML, err := buildPodPVCTemplateYAML(
		"default", "tmp", "topolvm-provisioner", "Filesystem", "tmp", "1Gi", "", "", "", "", "", "", "", nil)
	Expect(err).ShouldNot(HaveOccurred())
	Eventually(func(g Gomega) {
		stdout, stderr, err := kubectlWithInput(podPVCYAML, "apply", "-f", "-")
		g.Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s yaml=\n%s", stdout, stderr, podPVCYAML)
	}).Should(Succeed())
	Eventually(func(g Gomega) {
		stdout, stderr, err := kubectlWithInput(podPVCYAML, "delete", "-f", "-")
		g.Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
	}).Should(Succeed())

	By("[BeforeSuite] Creating namespace for test")
	createNamespace(testNamespace)
	createNamespace(testNamespace2)
})

var _ = AfterSuite(func() {
	if !failedTest {
		By("[AfterSuite] Delete namespace for autoresizer tests")
		stdout, stderr, err := kubectl("delete", "namespace", testNamespace)
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		stdout, stderr, err = kubectl("delete", "namespace", testNamespace2)
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
	}
})

type resource struct {
	resource string
	name     string
}

var _ = Describe("pvc-autoresizer", func() {
	var resources []resource
	var resources2 []resource

	_ = AfterEach(func() {
		if CurrentSpecReport().Failed() {
			failedTest = true
		} else {
			By("[AfterEach] cleanup resources")
			for _, r := range resources {
				stdout, stderr, err := kubectl("-n", testNamespace, "delete", "--ignore-not-found", r.resource, r.name)
				Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
			}
			for _, r := range resources2 {
				stdout, stderr, err := kubectl("-n", testNamespace2, "delete", "--ignore-not-found", r.resource, r.name)
				Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
			}
		}
		resources = []resource{}
		resources2 = []resource{}
	})

	It("should resize PVC based on PVC spec and annotations", func() {
		pvcName := "test-pvc"
		sc := "topolvm-provisioner-annotated"
		mode := string(corev1.PersistentVolumeFilesystem)
		request := "1Gi"
		threshold := "50%"
		increase := "1Gi"
		storageLimit := "2Gi"

		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request, threshold,
			"", increase, "", "", storageLimit, "", nil)

		By("create a file with a size that does not exceed threshold disk usage")
		stdout, stderr, err := kubectl("-n", testNamespace, "exec", pvcName, "--",
			"fallocate", "-l", "400M", "/test1/test1.txt")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk does not resize")
		checkDoesNotResize(pvcName, request)

		By("add a file with a size that exceed threshold disk usage")
		stdout, stderr, err = kubectl("-n", testNamespace, "exec", pvcName, "--",
			"fallocate", "-l", "200M", "/test1/test2.txt")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk resizing")
		checkDiskResize(pvcName, "2Gi", true)

		By("add a file with a size that exceed threshold disk usage too")
		stdout, stderr, err = kubectl("-n", testNamespace, "exec", pvcName, "--",
			"fallocate", "-l", "1G", "/test1/test3.txt")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk is not resized when the PVC size reaches the limit")
		checkDoesNotResize(pvcName, "2Gi")
	})

	It("should resize PVC when min-increase is less than increase by increase", func() {
		pvcName := "test-pvc-smaller-min"
		sc := "topolvm-provisioner-annotated"
		mode := string(corev1.PersistentVolumeFilesystem)
		request := "1Gi"
		threshold := "50%"
		increase := "2Gi"
		minimumIncrease := "1Gi"
		storageLimit := "3Gi"

		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request, threshold,
			"", increase, minimumIncrease, "", storageLimit, "", nil)

		By("create a file that exceeds threshold of disk usage")
		stdout, stderr, err := kubectl("-n", testNamespace, "exec", pvcName, "--",
			"fallocate", "-l", "600M", "/test1/test1.txt")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk resizing")
		checkDiskResize(pvcName, "3Gi", true)
	})

	It("should resize PVC when min-increase is greater than increase by min-increase", func() {
		pvcName := "test-pvc-larger-min"
		sc := "topolvm-provisioner-annotated"
		mode := string(corev1.PersistentVolumeFilesystem)
		request := "1Gi"
		threshold := "50%"
		increase := "1Gi"
		minimumIncrease := "2Gi"
		storageLimit := "3Gi"

		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request, threshold,
			"", increase, minimumIncrease, "", storageLimit, "", nil)

		By("create a file that exceeds threshold of disk usage")
		stdout, stderr, err := kubectl("-n", testNamespace, "exec", pvcName, "--",
			"fallocate", "-l", "600M", "/test1/test1.txt")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk resizing")
		checkDiskResize(pvcName, "3Gi", true)
	})

	It("should resize PVC no more than max-increase", func() {
		pvcName := "test-pvc-small-max"
		sc := "topolvm-provisioner-annotated"
		mode := string(corev1.PersistentVolumeFilesystem)
		request := "1Gi"
		threshold := "50%"
		increase := "2Gi"
		maximumIncrease := "1Gi"
		storageLimit := "3Gi"

		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request, threshold,
			"", increase, "", maximumIncrease, storageLimit, "", nil)

		By("create a file that exceeds threshold of disk usage")
		stdout, stderr, err := kubectl("-n", testNamespace, "exec", pvcName, "--",
			"fallocate", "-l", "600M", "/test1/test1.txt")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk resizing")
		checkDiskResize(pvcName, "2Gi", true)
	})

	It("should resize PVC based on pvc-autoresizer default setting", func() {
		pvcName := "test-pvc-default-setting"
		sc := "topolvm-provisioner-annotated"
		mode := string(corev1.PersistentVolumeFilesystem)
		request := "10Gi"
		threshold := ""
		increase := ""
		storageLimit := "11Gi"

		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request, threshold,
			"", increase, "", "", storageLimit, "", nil)

		By("create a file with a size that does not exceed threshold disk usage")
		stdout, stderr, err := kubectl("-n", testNamespace, "exec", pvcName, "--",
			"fallocate", "-l", "9G", "/test1/test1.txt")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk resizing")
		checkDiskResize(pvcName, "11Gi", false)
	})

	It("should not resize PVC with Block mode", func() {
		pvcName := "test-pvc-block-mode"
		sc := "topolvm-provisioner-annotated"
		mode := string(corev1.PersistentVolumeBlock)
		request := "1Gi"
		threshold := "50%"
		increase := "1Gi"
		storageLimit := "2Gi"

		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request, threshold,
			"", increase, "", "", storageLimit, "", nil)

		By("write data with a size that exceed threshold disk usage")
		stdout, stderr, err := kubectl("-n", testNamespace, "exec", pvcName, "--",
			"dd", "if=/dev/zero", "of=/dev/e2etest", "count=600M", "iflag=count_bytes")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk does not resize")
		checkDoesNotResize(pvcName, "1Gi")
	})

	It("should not resize PVC without available StorageClass", func() {
		pvcName := "test-pvc-with-disable-pvc-autoresizer"
		sc := "topolvm-provisioner"
		mode := string(corev1.PersistentVolumeFilesystem)
		request := "1Gi"
		threshold := "50%"
		increase := "1Gi"
		storageLimit := "2Gi"

		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request, threshold,
			"", increase, "", "", storageLimit, "", nil)

		By("create a file with a size that exceed threshold disk usage")
		stdout, stderr, err := kubectl("-n", testNamespace, "exec", pvcName, "--",
			"fallocate", "-l", "600M", "/test1/test1.txt")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk does not resize")
		checkDoesNotResize(pvcName, "1Gi")
	})

	It("should resize PVC based on PVC spec and annotations(inode)", func() {
		pvcName := "test-pvc"
		sc := "topolvm-provisioner-annotated"
		mode := string(corev1.PersistentVolumeFilesystem)
		request := "1Gi"
		threshold := "50%"
		increase := "1Gi"
		inodesThreshold := "99%"
		storageLimit := "2Gi"

		var capacityInode int64
		var availableInode int64

		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request, threshold, inodesThreshold, increase, "", "",
			storageLimit, "", nil)

		By("getting available inode size and capacity inode size")
		stdout, stderr, err := kubectl("-n", testNamespace, "exec", pvcName, "--",
			"df", "/test1", "--output=target,itotal,iavail")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		lines := regexp.MustCompile(`\n`).Split(string(stdout), -1)
		Expect(len(lines)).Should(Equal(3))

		cs := regexp.MustCompile(`\s+`).Split(lines[1], -1)
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
		stdout, stderr, err = kubectl("-n", testNamespace, "exec", pvcName, "--",
			"bash", "-c", fmt.Sprintf("touch /test1/testfile_{0..%d}.txt", num))
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk resizing")
		checkDiskResize(pvcName, "2Gi", true)
	})

	It("should not resize PVC when the storage limit is zero", func() {
		pvcName := "test-pvc-storage-limit-zero"
		sc := "topolvm-provisioner-annotated"
		mode := string(corev1.PersistentVolumeFilesystem)
		request := "3Gi"
		threshold := "50%"
		increase := "1Gi"
		storageLimit := "0Gi"

		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request, threshold,
			"", increase, "", "", storageLimit, "", nil)

		By("create a file with a size that does not exceed threshold disk usage")
		stdout, stderr, err := kubectl("-n", testNamespace, "exec", pvcName, "--",
			"fallocate", "-l", "2G", "/test1/test1.txt")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("checking the disk does not resize")
		checkDoesNotResize(pvcName, request)
	})

	It("should mutate the PVC size based on the same initial-resize-group-by PVC spec in the same namespace", func() {
		// large size PVC
		pvcName := "resize-group-pvc1"
		sc := "topolvm-provisioner-annotated"
		mode := string(corev1.PersistentVolumeFilesystem)
		request := "3Gi"
		threshold := "50%"
		increase := "1Gi"
		storageLimit := "10Gi"
		initialResizeGroupByAnnotation := "test-group"
		groupXLabel := map[string]string{
			initialResizeGroupByAnnotation: "group-x",
		}
		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request,
			threshold, "", increase, "", "", storageLimit, initialResizeGroupByAnnotation, groupXLabel)

		// large size PVC but other namespace
		pvcName = "resize-group-pvc2"
		request = "4Gi"
		storageLimit = "10Gi"
		resources2 = createPodPVC2(resources2, pvcName, sc, mode, pvcName,
			request, threshold, "", increase, "", "", storageLimit, initialResizeGroupByAnnotation, groupXLabel)

		// large size PVC but other group
		pvcName = "resize-group-pvc3"
		request = "4Gi"
		storageLimit = "10Gi"
		groupYLabel := map[string]string{
			initialResizeGroupByAnnotation: "group-y",
		}
		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request,
			threshold, "", increase, "", "", storageLimit, initialResizeGroupByAnnotation, groupYLabel)

		// Newly created small PVC
		pvcName = "resize-group-pvc4"
		request = "1Gi"
		storageLimit = "10Gi"
		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request,
			threshold, "", increase, "", "", storageLimit, initialResizeGroupByAnnotation, groupXLabel)

		By("checking the PVC size is mutated")
		checkDiskResize(pvcName, "3Gi", true)

		// Newly created small PVC but the storage limit is 2GiB
		pvcName = "resize-group-pvc5"
		request = "1Gi"
		storageLimit = "2Gi"
		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request,
			threshold, "", increase, "", "", storageLimit, initialResizeGroupByAnnotation, groupXLabel)

		By("checking the PVC size is mutated up to the storage limit")
		checkDiskResize(pvcName, "2Gi", true)
	})

	It("should not mutate the PVC size when the condition is not met", func() {
		// large size PVC
		pvcName := "non-resize-group-pvc1"
		sc := "topolvm-provisioner-annotated"
		mode := string(corev1.PersistentVolumeFilesystem)
		request := "3Gi"
		threshold := "50%"
		increase := "1Gi"
		storageLimit := "10Gi"
		initialResizeGroupByAnnotation := "test-group"
		groupXLabel := map[string]string{
			initialResizeGroupByAnnotation: "group-x",
		}
		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request,
			threshold, "", increase, "", "", storageLimit, initialResizeGroupByAnnotation, groupXLabel)

		// Newly created small PVC but different label key
		pvcName = "non-resize-group-pvc2"
		request = "1Gi"
		storageLimit = "10Gi"
		initialResizeGroupByAnnotation = "test-group2"
		groupXLabel2 := map[string]string{
			initialResizeGroupByAnnotation: "group-x",
		}
		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request,
			threshold, "", increase, "", "", storageLimit, initialResizeGroupByAnnotation, groupXLabel2)

		By("checking the PVC size is not mutated")
		checkDoesNotResize(pvcName, "1Gi")

		// The annotation is not set
		pvcName = "empty-string-resize-group-pvc1"
		request = "2Gi"
		storageLimit = "10Gi"
		initialResizeGroupByAnnotation = ""
		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request,
			threshold, "", increase, "", "", storageLimit, initialResizeGroupByAnnotation, groupXLabel)

		pvcName = "empty-string-resize-group-pvc2"
		request = "1Gi"
		storageLimit = "10Gi"
		resources = createPodPVC(resources, pvcName, sc, mode, pvcName, request,
			threshold, "", increase, "", "", storageLimit, initialResizeGroupByAnnotation, groupXLabel)

		By("checking the PVC size is not mutated")
		checkDoesNotResize(pvcName, "1Gi")
	})
})

func buildPodPVCTemplateYAML(ns, pvcName, storageClassName, volumeMode, podName, request, threshold, inodesThreshold,
	increase, minimumIncrease, maximumIncrease, storageLimit, initialResizeGroupByAnnotation string,
	labels map[string]string,
) ([]byte, error) {
	var b bytes.Buffer
	var err error

	useAnnotation := "false"
	if threshold != "" || increase != "" || storageLimit != "" {
		useAnnotation = "true"
	}
	useLabel := false
	if len(labels) != 0 {
		useLabel = true
	}

	podPVCTemplateOnce.Do(func() {
		podPVCTmpl, err = template.New("").Parse(podPVCTemplateYAML)
	})
	if err != nil {
		return b.Bytes(), err
	}

	params := map[string]any{
		"pvcName":                        pvcName,
		"storageClassName":               storageClassName,
		"volumeMode":                     volumeMode,
		"podName":                        podName,
		"namespace":                      ns,
		"useAnnotation":                  useAnnotation,
		"thresholdAnnotation":            threshold,
		"increaseAnnotation":             increase,
		"minimumIncreaseAnnotation":      minimumIncrease,
		"maximumIncreaseAnnotation":      maximumIncrease,
		"inodesThresholdAnnotation":      inodesThreshold,
		"storageLimitAnnotation":         storageLimit,
		"resourceRequest":                request,
		"initialResizeGroupByAnnotation": initialResizeGroupByAnnotation,
		"useLabel":                       useLabel,
		"labels":                         labels,
	}
	err = podPVCTmpl.Execute(&b, params)
	return b.Bytes(), err
}

func createPodPVC(resources []resource, pvcName, storageClassName, volumeMode, podName,
	request, threshold, inodesThreshold, increase, minimumIncrease, maximumincrease, storageLimit,
	initialResizeGroupByAnnotation string, labels map[string]string,
) []resource {
	return createPodPVCWithNamespace(testNamespace, resources, pvcName, storageClassName,
		volumeMode, podName, request, threshold, inodesThreshold, increase, minimumIncrease, maximumincrease, storageLimit,
		initialResizeGroupByAnnotation, labels)
}

func createPodPVC2(resources []resource, pvcName, storageClassName, volumeMode, podName,
	request, threshold, inodesThreshold, increase, minimumIncrease, maximumincrease, storageLimit,
	initialResizeGroupByAnnotation string, labels map[string]string,
) []resource {
	return createPodPVCWithNamespace(testNamespace2, resources, pvcName, storageClassName,
		volumeMode, podName, request, threshold, inodesThreshold, increase, minimumIncrease, maximumincrease, storageLimit,
		initialResizeGroupByAnnotation, labels)
}

func createPodPVCWithNamespace(ns string, resources []resource, pvcName, storageClassName,
	volumeMode, podName, request, threshold, inodesThreshold, increase, minimumIncrease, maximumincrease, storageLimit,
	initialResizeGroupByAnnotation string, labels map[string]string,
) []resource {
	By("create a PVC and a pod for test")
	podPVCYAML, err := buildPodPVCTemplateYAML(ns, pvcName, storageClassName, volumeMode, podName,
		request, threshold, inodesThreshold, increase, minimumIncrease, maximumincrease, storageLimit,
		initialResizeGroupByAnnotation, labels)
	Expect(err).ShouldNot(HaveOccurred())
	stdout, stderr, err := kubectlWithInput(podPVCYAML, "apply", "-f", "-")
	Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s yaml=\n%s", stdout, stderr, podPVCYAML)
	resources = append(resources, resource{resource: "pod", name: pvcName})
	resources = append(resources, resource{resource: "pvc", name: pvcName})

	By("waiting for creating the volume and running the pod")
	Eventually(func() error {
		stdout, stderr, err := kubectl("get", "-n", ns, "pod", pvcName, "-o", "yaml")
		if err != nil {
			return fmt.Errorf("failed to get pod name of %s/%s. stdout: %s, stderr: %s, err: %v",
				ns, pvcName, stdout, stderr, err)
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
