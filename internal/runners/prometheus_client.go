package runners

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/topolvm/pvc-autoresizer/internal/metrics"
	"k8s.io/apimachinery/pkg/types"
)

// NewPrometheusClient returns a new prometheusClient.
// labelMatchers is an optional comma-separated list of PromQL label matchers
// (e.g. `cluster="prod",env!="dev"`) added to every volume stats query. It is
// validated here so that an invalid value fails at startup instead of breaking
// every query at runtime.
func NewPrometheusClient(url string, labelMatchers string) (MetricsClient, error) {
	selector, err := parseLabelMatchers(labelMatchers)
	if err != nil {
		return nil, err
	}

	// kube-proxy pins each keep-alive connection to one backend, so after the Prometheus
	// Service is repointed a reused connection keeps reading the stale backend; disable
	// keep-alives to re-dial per query. Clone the default transport to keep its other fields.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableKeepAlives = true
	client, err := api.NewClient(api.Config{
		Address:      url,
		RoundTripper: transport,
	})
	if err != nil {
		return nil, err
	}
	v1api := prometheusv1.NewAPI(client)

	return &prometheusClient{
		prometheusAPI: v1api,
		selector:      selector,
	}, nil
}

// parseLabelMatchers parses comma-separated PromQL label matchers and returns
// them in canonical form, ready to be placed inside the braces of a vector
// selector. An empty input returns an empty string.
func parseLabelMatchers(labelMatchers string) (string, error) {
	if strings.TrimSpace(labelMatchers) == "" {
		return "", nil
	}
	matchers, err := parser.NewParser(parser.Options{}).ParseMetricSelector("{" + labelMatchers + "}")
	if err != nil {
		return "", fmt.Errorf("invalid prometheus label matchers %q: %w", labelMatchers, err)
	}
	parts := make([]string, 0, len(matchers))
	for _, m := range matchers {
		if m.Name == model.MetricNameLabel {
			return "", fmt.Errorf("invalid prometheus label matchers %q: matching on %s is not allowed",
				labelMatchers, model.MetricNameLabel)
		}
		parts = append(parts, m.String())
	}
	return strings.Join(parts, ","), nil
}

type prometheusClient struct {
	prometheusAPI prometheusv1.API
	// selector holds the validated label matchers, without braces.
	selector string
}

// GetMetrics implements MetricsClient.GetMetrics
func (c *prometheusClient) GetMetrics(ctx context.Context) (map[types.NamespacedName]*VolumeStats, error) {
	volumeStatsMap := make(map[types.NamespacedName]*VolumeStats)

	availableBytes, err := c.getMetricValues(ctx, c.buildQuery(volumeAvailableQuery))
	if err != nil {
		return nil, err
	}

	capacityBytes, err := c.getMetricValues(ctx, c.buildQuery(volumeCapacityQuery))
	if err != nil {
		return nil, err
	}

	availableInodeSize, err := c.getMetricValues(ctx, c.buildQuery(inodesAvailableQuery))
	if err != nil {
		return nil, err
	}

	capacityInodeSize, err := c.getMetricValues(ctx, c.buildQuery(inodesCapacityQuery))
	if err != nil {
		return nil, err
	}

	for key, val := range availableBytes {
		vs := &VolumeStats{AvailableBytes: val}
		if cb, ok := capacityBytes[key]; ok {
			vs.CapacityBytes = cb
		} else {
			continue
		}
		if ais, ok := availableInodeSize[key]; ok {
			vs.AvailableInodeSize = ais
		} else {
			continue
		}
		if cis, ok := capacityInodeSize[key]; ok {
			vs.CapacityInodeSize = cis
		} else {
			continue
		}
		volumeStatsMap[key] = vs
	}

	return volumeStatsMap, nil
}

// buildQuery returns the query for a metric name, restricted by the configured
// label matchers if any.
func (c *prometheusClient) buildQuery(metricName string) string {
	if c.selector == "" {
		return metricName
	}
	return metricName + "{" + c.selector + "}"
}

func (c *prometheusClient) getMetricValues(ctx context.Context, query string) (map[types.NamespacedName]int64, error) {
	res, _, err := c.prometheusAPI.Query(ctx, query, time.Now())
	if err != nil {
		metrics.MetricsClientFailTotal.Increment()
		return nil, err
	}

	if res.Type() != model.ValVector {
		return nil, fmt.Errorf("unknown response type: %s", res.Type().String())
	}
	resultMap := make(map[types.NamespacedName]int64)
	vec := res.(model.Vector)
	for _, val := range vec {
		nn := types.NamespacedName{
			Namespace: string(val.Metric["namespace"]),
			Name:      string(val.Metric["persistentvolumeclaim"]),
		}
		resultMap[nn] = int64(val.Value)
	}
	return resultMap, nil
}
