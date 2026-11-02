package runners

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/types"
)

func TestParseLabelMatchers(t *testing.T) {
	testCases := []struct {
		name          string
		labelMatchers string
		expected      string
		wantErr       bool
	}{
		{name: "empty", labelMatchers: "", expected: ""},
		{name: "whitespace only", labelMatchers: "  ", expected: ""},
		{name: "single matcher", labelMatchers: `cluster="prod"`, expected: `cluster="prod"`},
		{
			name:          "multiple matchers with every operator",
			labelMatchers: `cluster="prod", env!="dev",region=~"ap-.*",zone!~"z1"`,
			expected:      `cluster="prod",env!="dev",region=~"ap-.*",zone!~"z1"`,
		},
		{name: "unterminated quote", labelMatchers: `cluster="prod`, wantErr: true},
		{name: "missing value", labelMatchers: `cluster`, wantErr: true},
		{name: "braces included", labelMatchers: `{cluster="prod"}`, wantErr: true},
		{name: "metric name matcher", labelMatchers: `__name__="up"`, wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseLabelMatchers(tc.labelMatchers)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseLabelMatchers(%q) = %q, want error", tc.labelMatchers, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseLabelMatchers(%q) returned error: %v", tc.labelMatchers, err)
			}
			if got != tc.expected {
				t.Errorf("parseLabelMatchers(%q) = %q, want %q", tc.labelMatchers, got, tc.expected)
			}
		})
	}
}

func TestNewPrometheusClientRejectsInvalidLabelMatchers(t *testing.T) {
	if _, err := NewPrometheusClient("http://localhost:9090", `cluster=`); err == nil {
		t.Fatal("NewPrometheusClient with invalid label matchers succeeded, want error")
	}
}

func TestPrometheusClientGetMetricsWithLabelMatchers(t *testing.T) {
	testCases := []struct {
		name          string
		labelMatchers string
		querySuffix   string
	}{
		{name: "without label matchers", labelMatchers: "", querySuffix: ""},
		{name: "with label matchers", labelMatchers: `cluster="prod"`, querySuffix: `{cluster="prod"}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var queries []string
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := r.ParseForm(); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				mu.Lock()
				queries = append(queries, r.Form.Get("query"))
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, `{"status":"success","data":{"resultType":"vector","result":[`+
					`{"metric":{"namespace":"default","persistentvolumeclaim":"data"},"value":[0,"100"]}]}}`)
			}))
			defer ts.Close()

			c, err := NewPrometheusClient(ts.URL, tc.labelMatchers)
			if err != nil {
				t.Fatalf("NewPrometheusClient returned error: %v", err)
			}
			stats, err := c.GetMetrics(context.Background())
			if err != nil {
				t.Fatalf("GetMetrics returned error: %v", err)
			}
			if _, ok := stats[types.NamespacedName{Namespace: "default", Name: "data"}]; !ok {
				t.Errorf("GetMetrics result %v does not contain default/data", stats)
			}

			expected := []string{
				volumeAvailableQuery + tc.querySuffix,
				volumeCapacityQuery + tc.querySuffix,
				inodesAvailableQuery + tc.querySuffix,
				inodesCapacityQuery + tc.querySuffix,
			}
			if len(queries) != len(expected) {
				t.Fatalf("got queries %q, want %q", queries, expected)
			}
			for i := range expected {
				if queries[i] != expected[i] {
					t.Errorf("query %d = %q, want %q", i, queries[i], expected[i])
				}
			}
		})
	}
}
