package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/topolvm/pvc-autoresizer/internal/runners"
	"github.com/topolvm/pvc-autoresizer/internal/notifications"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	setupLog = ctrl.Log.WithName("setup")
)

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var useK8sMetricsAPI bool
	var prometheusURL string
	var namespaces string
	var interval time.Duration
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&useK8sMetricsAPI, "use-k8s-metrics-api", false,
		"Use Kubernetes metrics API instead of Prometheus.")
	flag.StringVar(&prometheusURL, "prometheus-url", "http://prometheus.monitoring.svc:9090",
		"Specify Prometheus URL to query volume stats.")
	flag.StringVar(&namespaces, "namespaces", "",
		"Specify namespaces to control the pvcs of. Empty for all namespaces.")
	flag.DurationVar(&interval, "interval", 10*time.Second,
		"Specify interval to monitor pvc capacity.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:              9443,
		LeaderElection:    enableLeaderElection,
		LeaderElectionID:  "pvc-autoresizer-leader",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Setup Slack configuration from environment variables
	slackConfig := &notifications.SlackConfig{
		WebhookURL: os.Getenv("SLACK_WEBHOOK_URL"),
		Channel:    os.Getenv("SLACK_CHANNEL"),
		Username:   os.Getenv("SLACK_USERNAME"),
		Enabled:    os.Getenv("SLACK_ENABLED") == "true",
	}

	// Send startup notification
	if slackConfig.Enabled {
		notifier := notifications.NewSlackNotifier(*slackConfig)
		clusterName := os.Getenv("CLUSTER_NAME")
		if clusterName == "" {
			clusterName = "current cluster" // Default if CLUSTER_NAME is not set
		}
		
		namespaceInfo := "all namespaces"
		if namespaces != "" {
			namespaceInfo = fmt.Sprintf("namespaces: %s", namespaces)
		}

		message := fmt.Sprintf("🚀 PVC Autoresizer started monitoring %s in %s\n"+
			"*Configuration:*\n"+
			"• Monitoring Interval: %s\n"+
			"• Using K8s Metrics API: %v\n"+
			"• Prometheus URL: %s",
			namespaceInfo,
			clusterName,
			interval,
			useK8sMetricsAPI,
			prometheusURL)

		if err := notifier.SendStartupNotification(message); err != nil {
			setupLog.Error(err, "failed to send startup notification")
		}
	}

	var metricsClient runners.MetricsClient
	if useK8sMetricsAPI {
		metricsClient = runners.NewK8sMetricsClient(mgr.GetClient())
	} else {
		metricsClient = runners.NewPrometheusClient(prometheusURL)
	}

	recorder := mgr.GetEventRecorderFor("pvc-autoresizer")

	if err = runners.SetupIndexer(mgr, false); err != nil {
		setupLog.Error(err, "unable to setup indexer")
		os.Exit(1)
	}

	if err = (&runners.PVCAutoresizer{}).SetupWithManager(mgr, metricsClient, interval, recorder, slackConfig); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PVCAutoresizer")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
} 