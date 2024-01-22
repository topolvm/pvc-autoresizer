package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	//+kubebuilder:scaffold:imports
)

var config struct {
	certDir        string
	webhookAddr    string
	metricsAddr    string
	healthAddr     string
	namespaces     []string
	watchInterval  time.Duration
	prometheusURL  string
	skipAnnotation bool
	development    bool
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pvc-autoresizer",
	Short: "PVC Autoresizer",
	Long: `pvc-autoresizer is an automatic volume resizer that edits PVCs if they have less than the specified ` +
		`amount of free filesystem capacity.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return subMain()
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	fs := rootCmd.Flags()
	fs.StringVar(&config.certDir, "cert-dir", "/certs", "webhook certificate directory")
	fs.StringVar(&config.webhookAddr, "webhook-addr", ":9443", "Listen address for the webhook endpoint")
	fs.StringVar(&config.metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	fs.StringVar(&config.healthAddr, "health-addr", ":8081", "The address of health/readiness probes.")
	fs.StringSliceVar(&config.namespaces, "namespaces", []string{},
		"Namespaces to resize PersistentVolumeClaims within. Empty for all namespaces.")
	fs.DurationVar(&config.watchInterval, "interval", 1*time.Minute, "Interval to monitor pvc capacity.")
	fs.StringVar(&config.prometheusURL, "prometheus-url", "", "Prometheus URL to query volume stats.")
	fs.BoolVar(&config.skipAnnotation, "no-annotation-check", false, "Skip annotation check for StorageClass")
	fs.BoolVar(&config.development, "development", false, "Use development logger config")
	_ = rootCmd.MarkFlagRequired("prometheus-url")
}
