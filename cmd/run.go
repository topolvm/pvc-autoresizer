package cmd

import (
	"net"
	"time"

	"github.com/topolvm/pvc-autoresizer/hooks"
	"github.com/topolvm/pvc-autoresizer/runners"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = corev1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func subMain() error {
	ctrl.SetLogger(zap.New(zap.UseDevMode(config.development)))

	var cacheFunc cache.NewCacheFunc
	if len(config.namespaces) > 0 {
		cacheFunc = cache.MultiNamespacedCacheBuilder(config.namespaces)
	}

	hookHost, portStr, err := net.SplitHostPort(config.webhookAddr)
	if err != nil {
		setupLog.Error(err, "invalid webhook addr")
		return err
	}
	hookPort, err := net.LookupPort("tcp", portStr)
	if err != nil {
		setupLog.Error(err, "invalid webhook port")
		return err
	}

	graceTimeout := 10 * time.Second
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		Host:                    hookHost,
		Port:                    hookPort,
		CertDir:                 config.certDir,
		MetricsBindAddress:      config.metricsAddr,
		NewCache:                cacheFunc,
		HealthProbeBindAddress:  config.healthAddr,
		LeaderElection:          true,
		LeaderElectionID:        "49e22f61.topolvm.io",
		GracefulShutdownTimeout: &graceTimeout,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		return err
	}
	if err := mgr.AddReadyzCheck("ping", healthz.Ping); err != nil {
		return err
	}
	if err := mgr.AddReadyzCheck("webhook", mgr.GetWebhookServer().StartedChecker()); err != nil {
		return err
	}

	promClient, err := runners.NewPrometheusClient(config.prometheusURL)
	if err != nil {
		setupLog.Error(err, "unable to initialize prometheus client")
		return err
	}

	if err := runners.SetupIndexer(mgr, config.skipAnnotation); err != nil {
		setupLog.Error(err, "unable to initialize pvc autoresizer")
		return err
	}

	pvcAutoresizer := runners.NewPVCAutoresizer(promClient, mgr.GetClient(),
		ctrl.Log.WithName("pvc-autoresizer"),
		config.watchInterval, mgr.GetEventRecorderFor("pvc-autoresizer"))
	if err := mgr.Add(pvcAutoresizer); err != nil {
		setupLog.Error(err, "unable to add autoresier to manager")
		return err
	}

	dec, err := admission.NewDecoder(scheme)
	if err != nil {
		setupLog.Error(err, "unable to create admission decoder")
		return err
	}
	if err = hooks.SetupPersistentVolumeClaimWebhook(mgr, dec, ctrl.Log.WithName("hooks")); err != nil {
		setupLog.Error(err, "unable to create PersistentVolumeClaim webhook")
		return err
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}
	return nil
}
