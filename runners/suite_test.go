package runners

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var cancelMgr func()
var promClient = prometheusClientMock{}

var scName string = "test-storageclass"
var provName string = "test-provisioner"

func TestRunners(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	scheme := runtime.NewScheme()
	err = corev1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = storagev1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: ":8080",
	})
	Expect(err).ToNot(HaveOccurred())

	noCheck := os.Getenv("NO_ANNOTATION_CHECK") == "true"
	err = SetupIndexer(mgr, noCheck)
	Expect(err).ToNot(HaveOccurred())

	pvcAutoresizer := NewPVCAutoresizer(&promClient, mgr.GetClient(),
		logf.Log.WithName("pvc-autoresizer"),
		1*time.Second, mgr.GetEventRecorderFor("pvc-autoresizer"))
	err = mgr.Add(pvcAutoresizer)
	Expect(err).ToNot(HaveOccurred())

	// Add pvcAutoresizer with FakeClientWrapper for metrics tests
	pvcAutoresizer2 := NewPVCAutoresizer(&promClient, NewFakeClientWrapper(mgr.GetClient()),
		logf.Log.WithName("pvc-autoresizer2"),
		1*time.Second, mgr.GetEventRecorderFor("pvc-autoresizer2"))
	err = mgr.Add(pvcAutoresizer2)
	Expect(err).ToNot(HaveOccurred())

	ctx, cancel := context.WithCancel(context.Background())
	cancelMgr = cancel
	go func() {
		err = mgr.Start(ctx)
		if err != nil {
			mgr.GetLogger().Error(err, "failed to start manager")
		}
	}()

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	createStorageClass(ctx, scName, provName)

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancelMgr()
	time.Sleep(10 * time.Millisecond)
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func createStorageClass(ctx context.Context, name, provisioner string) {
	t := true
	sc := storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				AutoResizeEnabledKey: "true",
			},
		},
		Provisioner:          provisioner,
		AllowVolumeExpansion: &t,
	}
	err := k8sClient.Create(ctx, &sc)
	Expect(err).NotTo(HaveOccurred())
}

func getMetricsFamily() (map[string]*dto.MetricFamily, error) {
	resp, err := http.Get("http://localhost:8080/metrics")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parser expfmt.TextParser
	return parser.TextToMetricFamilies(resp.Body)
}

func haveLabels(m *dto.Metric, labels map[string]string) bool {
OUTER:
	for k, v := range labels {
		for _, label := range m.Label {
			if k == *label.Name && v == *label.Value {
				continue OUTER
			}
		}
		return false
	}
	return true
}
