package main

import (
	"net"
	"time"

	"github.com/topolvm/pvc-autoresizer/hooks"
	"github.com/topolvm/pvc-autoresizer/runners"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(corev1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func subMain() error {
	ctrl.SetLogger(zap.New(zap.UseDevMode(config.development)))

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

	var pvcCacheTarget cache.ByObject
	if len(config.namespaces) == 0 {
		pvcCacheTarget = cache.ByObject{
			Namespaces: map[string]cache.Config{
				cache.AllNamespaces: {},
			},
		}
	} else {
		pvcCacheTarget = cache.ByObject{
			Namespaces: map[string]cache.Config{},
		}
		for _, ns := range config.namespaces {
			pvcCacheTarget.Namespaces[ns] = cache.Config{}
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    hookHost,
			Port:    hookPort,
			CertDir: config.certDir,
		}),
		Metrics: metricsserver.Options{
			BindAddress: config.metricsAddr,
		},
		Cache: cache.Options{
			ByObject: map[client.Object]cache.ByObject{
				&corev1.PersistentVolumeClaim{}: pvcCacheTarget,
				&storagev1.StorageClass{}:       {},
			},
		},
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

	dec := admission.NewDecoder(scheme)
	if err = hooks.SetupPersistentVolumeClaimWebhook(mgr, dec, ctrl.Log.WithName("hooks")); err != nil {
		setupLog.Error(err, "unable to create PersistentVolumeClaim webhook")
		return err
	}

	//+kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}
	return nil
}
