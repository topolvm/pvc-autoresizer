package main

import (
	"errors"
	"flag"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/topolvm/pvc-autoresizer/runners"
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

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var watchInterval time.Duration
	var prometheusURL string
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.DurationVar(&watchInterval, "interval", 10*time.Second, "Interval to monitor pvc capacity.")
	flag.StringVar(&prometheusURL, "prometheus-url", "", "Prometheus URL to query volume stats.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	if prometheusURL == "" {
		setupLog.Error(errors.New("prometheus-url is empty"), "prometheus-url is required")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "49e22f61.topolvm.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	promClient, err := runners.NewPrometheusClient(prometheusURL)
	if err != nil {
		setupLog.Error(err, "unable to initialize prometheus client")
		os.Exit(1)
	}

	pvcAutoresizer := runners.NewPVCAutoresizer(promClient, watchInterval, mgr.GetEventRecorderFor("pvc-autoresizer"))
	err = pvcAutoresizer.SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to initialize pvc autoresizer")
		os.Exit(1)
	}
	err = mgr.Add(pvcAutoresizer)
	if err != nil {
		setupLog.Error(err, "unable to add autoresier to manager")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
