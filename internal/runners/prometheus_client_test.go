package runners

import "testing"

func TestPrometheusClientBuildQuery(t *testing.T) {
	testCases := []struct {
		name          string
		labelSelector string
		metricName    string
		expected      string
	}{
		{
			name:          "no selector returns bare metric name",
			labelSelector: "",
			metricName:    volumeAvailableQuery,
			expected:      `kubelet_volume_stats_available_bytes`,
		},
		{
			name:          "single matcher",
			labelSelector: `cluster="prod"`,
			metricName:    volumeAvailableQuery,
			expected:      `kubelet_volume_stats_available_bytes{cluster="prod"}`,
		},
		{
			name:          "multiple matchers",
			labelSelector: `cluster="prod",env!="dev"`,
			metricName:    inodesCapacityQuery,
			expected:      `kubelet_volume_stats_inodes{cluster="prod",env!="dev"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := &prometheusClient{labelSelector: tc.labelSelector}
			got := c.buildQuery(tc.metricName)
			if got != tc.expected {
				t.Errorf("buildQuery(%q) = %q, want %q", tc.metricName, got, tc.expected)
			}
		})
	}
}
