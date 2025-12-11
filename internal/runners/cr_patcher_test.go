package runners

import (
	"testing"

	pvcautoresizer "github.com/topolvm/pvc-autoresizer"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseCRTargetAnnotations_NoAnnotations(t *testing.T) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
		},
	}

	config, err := parseCRTargetAnnotations(pvc)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if config != nil {
		t.Errorf("expected nil config, got %+v", config)
	}
}

func TestParseCRTargetAnnotations_Valid(t *testing.T) {
	tests := []struct {
		name              string
		annotations       map[string]string
		expectedNamespace string
		expectedPath      string
	}{
		{
			name: "all required annotations with namespace",
			annotations: map[string]string{
				pvcautoresizer.TargetResourceAPIVersionAnnotation: "rabbitmq.com/v1beta1",
				pvcautoresizer.TargetResourceKindAnnotation:       "RabbitmqCluster",
				pvcautoresizer.TargetResourceNameAnnotation:       "my-rabbitmq",
				pvcautoresizer.TargetResourceNamespaceAnnotation:  "rabbitmq-system",
				pvcautoresizer.TargetResourceJSONPathAnnotation:   ".spec.persistence.storage",
			},
			expectedNamespace: "rabbitmq-system",
			expectedPath:      "/spec/persistence/storage",
		},
		{
			name: "namespace defaults to PVC namespace",
			annotations: map[string]string{
				pvcautoresizer.TargetResourceAPIVersionAnnotation: "postgresql.cnpg.io/v1",
				pvcautoresizer.TargetResourceKindAnnotation:       "Cluster",
				pvcautoresizer.TargetResourceNameAnnotation:       "my-cluster",
				pvcautoresizer.TargetResourceJSONPathAnnotation:   ".spec.storage.size",
			},
			expectedNamespace: "default",
			expectedPath:      "/spec/storage/size",
		},
		{
			name: "JSON path already in pointer format",
			annotations: map[string]string{
				pvcautoresizer.TargetResourceAPIVersionAnnotation: "dragonflydb.io/v1alpha1",
				pvcautoresizer.TargetResourceKindAnnotation:       "Dragonfly",
				pvcautoresizer.TargetResourceNameAnnotation:       "my-dragonfly",
				pvcautoresizer.TargetResourceJSONPathAnnotation:   "/spec/resources/requests/storage",
			},
			expectedNamespace: "default",
			expectedPath:      "/spec/resources/requests/storage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-pvc",
					Namespace:   "default",
					Annotations: tt.annotations,
				},
			}

			config, err := parseCRTargetAnnotations(pvc)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if config == nil {
				t.Fatal("expected config, got nil")
			}

			if config.APIVersion != tt.annotations[pvcautoresizer.TargetResourceAPIVersionAnnotation] {
				t.Errorf("expected APIVersion %q, got %q",
					tt.annotations[pvcautoresizer.TargetResourceAPIVersionAnnotation], config.APIVersion)
			}
			if config.Kind != tt.annotations[pvcautoresizer.TargetResourceKindAnnotation] {
				t.Errorf("expected Kind %q, got %q",
					tt.annotations[pvcautoresizer.TargetResourceKindAnnotation], config.Kind)
			}
			if config.Name != tt.annotations[pvcautoresizer.TargetResourceNameAnnotation] {
				t.Errorf("expected Name %q, got %q",
					tt.annotations[pvcautoresizer.TargetResourceNameAnnotation], config.Name)
			}
			if config.Namespace != tt.expectedNamespace {
				t.Errorf("expected Namespace %q, got %q", tt.expectedNamespace, config.Namespace)
			}
			if config.JSONPath != tt.expectedPath {
				t.Errorf("expected JSONPath %q, got %q", tt.expectedPath, config.JSONPath)
			}
		})
	}
}

func TestParseCRTargetAnnotations_MissingRequired(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		wantErr     bool
	}{
		{
			name: "missing api-version",
			annotations: map[string]string{
				pvcautoresizer.TargetResourceKindAnnotation:     "RabbitmqCluster",
				pvcautoresizer.TargetResourceNameAnnotation:     "my-rabbitmq",
				pvcautoresizer.TargetResourceJSONPathAnnotation: ".spec.persistence.storage",
			},
			wantErr: true,
		},
		{
			name: "missing kind",
			annotations: map[string]string{
				pvcautoresizer.TargetResourceAPIVersionAnnotation: "rabbitmq.com/v1beta1",
				pvcautoresizer.TargetResourceNameAnnotation:       "my-rabbitmq",
				pvcautoresizer.TargetResourceJSONPathAnnotation:   ".spec.persistence.storage",
			},
			wantErr: true,
		},
		{
			name: "missing name",
			annotations: map[string]string{
				pvcautoresizer.TargetResourceAPIVersionAnnotation: "rabbitmq.com/v1beta1",
				pvcautoresizer.TargetResourceKindAnnotation:       "RabbitmqCluster",
				pvcautoresizer.TargetResourceJSONPathAnnotation:   ".spec.persistence.storage",
			},
			wantErr: true,
		},
		{
			name: "missing json-path",
			annotations: map[string]string{
				pvcautoresizer.TargetResourceAPIVersionAnnotation: "rabbitmq.com/v1beta1",
				pvcautoresizer.TargetResourceKindAnnotation:       "RabbitmqCluster",
				pvcautoresizer.TargetResourceNameAnnotation:       "my-rabbitmq",
			},
			wantErr: true,
		},
		{
			name: "only one annotation present",
			annotations: map[string]string{
				pvcautoresizer.TargetResourceKindAnnotation: "RabbitmqCluster",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-pvc",
					Namespace:   "default",
					Annotations: tt.annotations,
				},
			}

			config, err := parseCRTargetAnnotations(pvc)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if config != nil {
					t.Errorf("expected nil config on error, got %+v", config)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestNormalizeJSONPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    ".spec.persistence.storage",
			expected: "/spec/persistence/storage",
		},
		{
			input:    "spec.persistence.storage",
			expected: "/spec/persistence/storage",
		},
		{
			input:    "/spec/persistence/storage",
			expected: "/spec/persistence/storage",
		},
		{
			input:    ".spec.storage.size",
			expected: "/spec/storage/size",
		},
		{
			input:    "spec",
			expected: "/spec",
		},
		{
			input:    ".spec",
			expected: "/spec",
		},
		{
			input:    "/spec",
			expected: "/spec",
		},
		{
			input:    ".spec.resources.requests.storage",
			expected: "/spec/resources/requests/storage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeJSONPath(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeJSONPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateJSONPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		// Valid paths - must start with /spec/ and target a specific field
		{name: "valid spec path", path: ".spec.storage.size", wantErr: false},
		{name: "valid spec nested", path: "/spec/persistence/storage", wantErr: false},
		{name: "valid spec deep", path: ".spec.resources.requests.storage", wantErr: false},
		{name: "valid without leading dot", path: "spec.persistence.storage", wantErr: false},
		{name: "valid pointer format", path: "/spec/storage/size", wantErr: false},

		// Invalid paths - security violations
		{name: "metadata blocked", path: ".metadata.annotations.foo", wantErr: true},
		{name: "status blocked", path: "/status/conditions", wantErr: true},
		{name: "metadata labels blocked", path: ".metadata.labels.app", wantErr: true},
		{name: "metadata ownerRefs blocked", path: "/metadata/ownerReferences", wantErr: true},
		{name: "spec root only", path: "/spec", wantErr: true},
		{name: "spec root with dot", path: ".spec", wantErr: true},
		{name: "spec root with trailing slash", path: "/spec/", wantErr: true},
		{name: "non-spec root", path: "/apiVersion", wantErr: true},
		{name: "kind field blocked", path: ".kind", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJSONPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateJSONPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestParseAPIVersion(t *testing.T) {
	tests := []struct {
		name        string
		apiVersion  string
		wantGroup   string
		wantVersion string
		wantErr     bool
	}{
		{
			name:        "custom resource with group",
			apiVersion:  "rabbitmq.com/v1beta1",
			wantGroup:   "rabbitmq.com",
			wantVersion: "v1beta1",
			wantErr:     false,
		},
		{
			name:        "core API group",
			apiVersion:  "v1",
			wantGroup:   "",
			wantVersion: "v1",
			wantErr:     false,
		},
		{
			name:        "postgresql cnpg",
			apiVersion:  "postgresql.cnpg.io/v1",
			wantGroup:   "postgresql.cnpg.io",
			wantVersion: "v1",
			wantErr:     false,
		},
		{
			name:        "dragonflydb",
			apiVersion:  "dragonflydb.io/v1alpha1",
			wantGroup:   "dragonflydb.io",
			wantVersion: "v1alpha1",
			wantErr:     false,
		},
		{
			name:       "empty string",
			apiVersion: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group, version, err := parseAPIVersion(tt.apiVersion)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if group != tt.wantGroup {
				t.Errorf("expected group %q, got %q", tt.wantGroup, group)
			}
			if version != tt.wantVersion {
				t.Errorf("expected version %q, got %q", tt.wantVersion, version)
			}
		})
	}
}
