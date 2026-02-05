package runners

import (
	"testing"

	pvcautoresizer "github.com/topolvm/pvc-autoresizer"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseCRTargetAnnotations_ValidClass(t *testing.T) {
	resourceClasses := map[string]ResourceClass{
		"rabbitmq": {
			Name:       "rabbitmq",
			APIGroup:   "rabbitmq.com",
			APIVersion: "v1beta1",
			Kind:       "RabbitmqCluster",
			Path:       "/spec/persistence/storage",
		},
		"cnpg-data": {
			Name:       "cnpg-data",
			APIGroup:   "postgresql.cnpg.io",
			APIVersion: "v1",
			Kind:       "Cluster",
			Path:       "/spec/storage/size",
		},
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
			Annotations: map[string]string{
				pvcautoresizer.TargetResourceClassAnnotation: "rabbitmq",
				pvcautoresizer.TargetResourceNameAnnotation:  "my-rabbitmq",
			},
		},
	}

	config, err := parseCRTargetAnnotations(pvc, resourceClasses)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config == nil {
		t.Fatal("expected config, got nil")
	}

	if config.APIVersion != "rabbitmq.com/v1beta1" {
		t.Errorf("expected APIVersion 'rabbitmq.com/v1beta1', got %q", config.APIVersion)
	}
	if config.Kind != "RabbitmqCluster" {
		t.Errorf("expected Kind 'RabbitmqCluster', got %q", config.Kind)
	}
	if config.Name != "my-rabbitmq" {
		t.Errorf("expected Name 'my-rabbitmq', got %q", config.Name)
	}
	if config.Namespace != "default" {
		t.Errorf("expected Namespace 'default', got %q", config.Namespace)
	}
	if config.JSONPath != "/spec/persistence/storage" {
		t.Errorf("expected JSONPath '/spec/persistence/storage', got %q", config.JSONPath)
	}
}

func TestParseCRTargetAnnotations_WithExplicitNamespace(t *testing.T) {
	resourceClasses := map[string]ResourceClass{
		"rabbitmq": {
			Name:       "rabbitmq",
			APIGroup:   "rabbitmq.com",
			APIVersion: "v1beta1",
			Kind:       "RabbitmqCluster",
			Path:       "/spec/persistence/storage",
		},
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
			Annotations: map[string]string{
				pvcautoresizer.TargetResourceClassAnnotation: "rabbitmq",
				pvcautoresizer.TargetResourceNameAnnotation:  "my-rabbitmq",
			},
		},
	}

	config, err := parseCRTargetAnnotations(pvc, resourceClasses)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// CR namespace must match PVC namespace (cross-namespace not supported)
	if config.Namespace != "default" {
		t.Errorf("expected Namespace 'default' (same as PVC), got %q", config.Namespace)
	}
}

func TestParseCRTargetAnnotations_NoAnnotations(t *testing.T) {
	resourceClasses := map[string]ResourceClass{
		"rabbitmq": {
			Name:       "rabbitmq",
			APIGroup:   "rabbitmq.com",
			APIVersion: "v1beta1",
			Kind:       "RabbitmqCluster",
			Path:       "/spec/persistence/storage",
		},
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
		},
	}

	config, err := parseCRTargetAnnotations(pvc, resourceClasses)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if config != nil {
		t.Errorf("expected nil config, got %+v", config)
	}
}

func TestParseCRTargetAnnotations_UnknownClass(t *testing.T) {
	resourceClasses := map[string]ResourceClass{
		"rabbitmq": {
			Name:       "rabbitmq",
			APIGroup:   "rabbitmq.com",
			APIVersion: "v1beta1",
			Kind:       "RabbitmqCluster",
			Path:       "/spec/persistence/storage",
		},
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
			Annotations: map[string]string{
				pvcautoresizer.TargetResourceClassAnnotation: "unknown-class",
				pvcautoresizer.TargetResourceNameAnnotation:  "my-resource",
			},
		},
	}

	_, err := parseCRTargetAnnotations(pvc, resourceClasses)
	if err == nil {
		t.Fatal("expected error for unknown class, got nil")
	}
	if !containsSubstring(err.Error(), "unknown resource class") {
		t.Errorf("expected error to mention 'unknown resource class', got: %v", err)
	}
}

func TestParseCRTargetAnnotations_MissingName(t *testing.T) {
	resourceClasses := map[string]ResourceClass{
		"rabbitmq": {
			Name:       "rabbitmq",
			APIGroup:   "rabbitmq.com",
			APIVersion: "v1beta1",
			Kind:       "RabbitmqCluster",
			Path:       "/spec/persistence/storage",
		},
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
			Annotations: map[string]string{
				pvcautoresizer.TargetResourceClassAnnotation: "rabbitmq",
				// Missing TargetResourceNameAnnotation
			},
		},
	}

	_, err := parseCRTargetAnnotations(pvc, resourceClasses)
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
	if !containsSubstring(err.Error(), "target-resource-name") {
		t.Errorf("expected error to mention 'target-resource-name', got: %v", err)
	}
}

func TestParseCRTargetAnnotations_EmptyResourceClasses(t *testing.T) {
	resourceClasses := map[string]ResourceClass{}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
			Annotations: map[string]string{
				pvcautoresizer.TargetResourceClassAnnotation: "rabbitmq",
				pvcautoresizer.TargetResourceNameAnnotation:  "my-rabbitmq",
			},
		},
	}

	_, err := parseCRTargetAnnotations(pvc, resourceClasses)
	if err == nil {
		t.Fatal("expected error when resource classes are empty, got nil")
	}
}

func TestParseCRTargetAnnotations_NilResourceClasses(t *testing.T) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
			Annotations: map[string]string{
				pvcautoresizer.TargetResourceClassAnnotation: "rabbitmq",
				pvcautoresizer.TargetResourceNameAnnotation:  "my-rabbitmq",
			},
		},
	}

	_, err := parseCRTargetAnnotations(pvc, nil)
	if err == nil {
		t.Fatal("expected error when resource classes are nil, got nil")
	}
}

func TestParseCRTargetAnnotations_DifferentClasses(t *testing.T) {
	resourceClasses := map[string]ResourceClass{
		"rabbitmq": {
			Name:       "rabbitmq",
			APIGroup:   "rabbitmq.com",
			APIVersion: "v1beta1",
			Kind:       "RabbitmqCluster",
			Path:       "/spec/persistence/storage",
		},
		"cnpg-data": {
			Name:       "cnpg-data",
			APIGroup:   "postgresql.cnpg.io",
			APIVersion: "v1",
			Kind:       "Cluster",
			Path:       "/spec/storage/size",
		},
		"cnpg-wal": {
			Name:       "cnpg-wal",
			APIGroup:   "postgresql.cnpg.io",
			APIVersion: "v1",
			Kind:       "Cluster",
			Path:       "/spec/walStorage/size",
		},
	}

	tests := []struct {
		name         string
		className    string
		resourceName string
		expectedPath string
		expectedKind string
	}{
		{
			name:         "rabbitmq class",
			className:    "rabbitmq",
			resourceName: "my-rabbitmq",
			expectedPath: "/spec/persistence/storage",
			expectedKind: "RabbitmqCluster",
		},
		{
			name:         "cnpg-data class",
			className:    "cnpg-data",
			resourceName: "my-postgres",
			expectedPath: "/spec/storage/size",
			expectedKind: "Cluster",
		},
		{
			name:         "cnpg-wal class",
			className:    "cnpg-wal",
			resourceName: "my-postgres",
			expectedPath: "/spec/walStorage/size",
			expectedKind: "Cluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pvc",
					Namespace: "default",
					Annotations: map[string]string{
						pvcautoresizer.TargetResourceClassAnnotation: tt.className,
						pvcautoresizer.TargetResourceNameAnnotation:  tt.resourceName,
					},
				},
			}

			config, err := parseCRTargetAnnotations(pvc, resourceClasses)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if config.JSONPath != tt.expectedPath {
				t.Errorf("expected JSONPath %q, got %q", tt.expectedPath, config.JSONPath)
			}
			if config.Kind != tt.expectedKind {
				t.Errorf("expected Kind %q, got %q", tt.expectedKind, config.Kind)
			}
			if config.Name != tt.resourceName {
				t.Errorf("expected Name %q, got %q", tt.resourceName, config.Name)
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

// Helper function
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Tests for path parsing with filter support

func TestParsePath_SimpleSegments(t *testing.T) {
	segments, err := parsePath("/spec/storage/size")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(segments) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(segments))
	}

	expected := []pathSegment{
		{Field: "spec", FilterKey: "", FilterVal: ""},
		{Field: "storage", FilterKey: "", FilterVal: ""},
		{Field: "size", FilterKey: "", FilterVal: ""},
	}

	for i, seg := range segments {
		if seg.Field != expected[i].Field {
			t.Errorf("segment %d: expected Field %q, got %q", i, expected[i].Field, seg.Field)
		}
		if seg.FilterKey != expected[i].FilterKey {
			t.Errorf("segment %d: expected FilterKey %q, got %q", i, expected[i].FilterKey, seg.FilterKey)
		}
		if seg.FilterVal != expected[i].FilterVal {
			t.Errorf("segment %d: expected FilterVal %q, got %q", i, expected[i].FilterVal, seg.FilterVal)
		}
	}
}

func TestParsePath_WithFilter(t *testing.T) {
	segments, err := parsePath("/spec/tablespaces[name=tbs1]/storage/size")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(segments) != 4 {
		t.Fatalf("expected 4 segments, got %d", len(segments))
	}

	// Check the segment with the filter
	if segments[1].Field != "tablespaces" {
		t.Errorf("expected Field 'tablespaces', got %q", segments[1].Field)
	}
	if segments[1].FilterKey != "name" {
		t.Errorf("expected FilterKey 'name', got %q", segments[1].FilterKey)
	}
	if segments[1].FilterVal != "tbs1" {
		t.Errorf("expected FilterVal 'tbs1', got %q", segments[1].FilterVal)
	}
}

func TestParsePath_WithPlaceholder(t *testing.T) {
	segments, err := parsePath("/spec/tablespaces[name=?]/storage/size")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(segments) != 4 {
		t.Fatalf("expected 4 segments, got %d", len(segments))
	}

	// Check the segment with the placeholder
	if segments[1].Field != "tablespaces" {
		t.Errorf("expected Field 'tablespaces', got %q", segments[1].Field)
	}
	if segments[1].FilterKey != "name" {
		t.Errorf("expected FilterKey 'name', got %q", segments[1].FilterKey)
	}
	if segments[1].FilterVal != "?" {
		t.Errorf("expected FilterVal '?', got %q", segments[1].FilterVal)
	}
}

func TestParsePath_InvalidFilterSyntax(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "missing closing bracket", path: "/spec/tablespaces[name=tbs1/storage"},
		{name: "missing equals", path: "/spec/tablespaces[name]/storage"},
		{name: "empty filter key", path: "/spec/tablespaces[=value]/storage"},
		{name: "empty brackets", path: "/spec/tablespaces[]/storage"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parsePath(tt.path)
			if err == nil {
				t.Errorf("expected error for path %q, got nil", tt.path)
			}
		})
	}
}

func TestParsePath_MultipleFilters(t *testing.T) {
	_, err := parsePath("/spec/tablespaces[name=tbs1]/volumes[id=vol1]/size")
	if err == nil {
		t.Error("expected error for multiple filters, got nil")
	}
	if !containsSubstring(err.Error(), "multiple filters") {
		t.Errorf("expected error to mention 'multiple filters', got: %v", err)
	}
}

func TestParsePath_EmptyPath(t *testing.T) {
	_, err := parsePath("")
	if err == nil {
		t.Error("expected error for empty path, got nil")
	}
}

func TestParsePath_RootOnly(t *testing.T) {
	_, err := parsePath("/")
	if err == nil {
		t.Error("expected error for root-only path, got nil")
	}
}

// Tests for placeholder resolution

func TestResolvePlaceholder_Success(t *testing.T) {
	segments := []pathSegment{
		{Field: "spec", FilterKey: "", FilterVal: ""},
		{Field: "tablespaces", FilterKey: "name", FilterVal: "?"},
		{Field: "storage", FilterKey: "", FilterVal: ""},
		{Field: "size", FilterKey: "", FilterVal: ""},
	}

	resolved, err := resolvePlaceholder(segments, "tbs1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved[1].FilterVal != "tbs1" {
		t.Errorf("expected FilterVal 'tbs1', got %q", resolved[1].FilterVal)
	}
	// Original should be unchanged
	if segments[1].FilterVal != "?" {
		t.Errorf("original was modified: expected '?', got %q", segments[1].FilterVal)
	}
}

func TestResolvePlaceholder_MissingAnnotation(t *testing.T) {
	segments := []pathSegment{
		{Field: "spec", FilterKey: "", FilterVal: ""},
		{Field: "tablespaces", FilterKey: "name", FilterVal: "?"},
		{Field: "storage", FilterKey: "", FilterVal: ""},
	}

	_, err := resolvePlaceholder(segments, "")
	if err == nil {
		t.Fatal("expected error for missing filter value, got nil")
	}
	if !containsSubstring(err.Error(), "target-filter-value") {
		t.Errorf("expected error to mention 'target-filter-value', got: %v", err)
	}
}

func TestResolvePlaceholder_NoPlaceholder(t *testing.T) {
	segments := []pathSegment{
		{Field: "spec", FilterKey: "", FilterVal: ""},
		{Field: "storage", FilterKey: "", FilterVal: ""},
		{Field: "size", FilterKey: "", FilterVal: ""},
	}

	// Should succeed even without annotation when there's no placeholder
	resolved, err := resolvePlaceholder(segments, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Segments should be unchanged
	if len(resolved) != len(segments) {
		t.Errorf("expected %d segments, got %d", len(segments), len(resolved))
	}
}

func TestResolvePlaceholder_HardcodedFilter(t *testing.T) {
	segments := []pathSegment{
		{Field: "spec", FilterKey: "", FilterVal: ""},
		{Field: "tablespaces", FilterKey: "name", FilterVal: "tbs1"},
		{Field: "storage", FilterKey: "", FilterVal: ""},
	}

	// Should succeed and leave hardcoded value unchanged
	resolved, err := resolvePlaceholder(segments, "ignored-value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved[1].FilterVal != "tbs1" {
		t.Errorf("expected FilterVal 'tbs1' (unchanged), got %q", resolved[1].FilterVal)
	}
}

// Tests for array navigation (setNestedFieldWithFilter)

func TestSetNestedFieldWithFilter_SimpleArray(t *testing.T) {
	obj := map[string]interface{}{
		"spec": map[string]interface{}{
			"tablespaces": []interface{}{
				map[string]interface{}{
					"name": "tbs1",
					"storage": map[string]interface{}{
						"size": "10Gi",
					},
				},
				map[string]interface{}{
					"name": "tbs2",
					"storage": map[string]interface{}{
						"size": "20Gi",
					},
				},
			},
		},
	}

	segments := []pathSegment{
		{Field: "spec"},
		{Field: "tablespaces", FilterKey: "name", FilterVal: "tbs1"},
		{Field: "storage"},
		{Field: "size"},
	}

	err := setNestedFieldWithFilter(obj, segments, "50Gi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the value was set
	spec := obj["spec"].(map[string]interface{})
	tablespaces := spec["tablespaces"].([]interface{})
	tbs1 := tablespaces[0].(map[string]interface{})
	storage := tbs1["storage"].(map[string]interface{})
	if storage["size"] != "50Gi" {
		t.Errorf("expected size '50Gi', got %q", storage["size"])
	}

	// Verify tbs2 was not modified
	tbs2 := tablespaces[1].(map[string]interface{})
	storage2 := tbs2["storage"].(map[string]interface{})
	if storage2["size"] != "20Gi" {
		t.Errorf("tbs2 was modified: expected size '20Gi', got %q", storage2["size"])
	}
}

func TestSetNestedFieldWithFilter_NoMatch(t *testing.T) {
	obj := map[string]interface{}{
		"spec": map[string]interface{}{
			"tablespaces": []interface{}{
				map[string]interface{}{
					"name": "tbs1",
					"storage": map[string]interface{}{
						"size": "10Gi",
					},
				},
			},
		},
	}

	segments := []pathSegment{
		{Field: "spec"},
		{Field: "tablespaces", FilterKey: "name", FilterVal: "nonexistent"},
		{Field: "storage"},
		{Field: "size"},
	}

	err := setNestedFieldWithFilter(obj, segments, "50Gi")
	// Should return nil (no error) but also not modify anything
	if err != nil {
		t.Fatalf("expected nil error for no match, got: %v", err)
	}
}

func TestSetNestedFieldWithFilter_MultipleMatches(t *testing.T) {
	obj := map[string]interface{}{
		"spec": map[string]interface{}{
			"tablespaces": []interface{}{
				map[string]interface{}{
					"name":    "tbs1",
					"storage": map[string]interface{}{"size": "10Gi"},
				},
				map[string]interface{}{
					"name":    "tbs1", // Duplicate!
					"storage": map[string]interface{}{"size": "20Gi"},
				},
			},
		},
	}

	segments := []pathSegment{
		{Field: "spec"},
		{Field: "tablespaces", FilterKey: "name", FilterVal: "tbs1"},
		{Field: "storage"},
		{Field: "size"},
	}

	err := setNestedFieldWithFilter(obj, segments, "50Gi")
	if err == nil {
		t.Fatal("expected error for multiple matches, got nil")
	}
	if !containsSubstring(err.Error(), "multiple") {
		t.Errorf("expected error to mention 'multiple', got: %v", err)
	}
}

func TestSetNestedFieldWithFilter_NotAnArray(t *testing.T) {
	obj := map[string]interface{}{
		"spec": map[string]interface{}{
			"tablespaces": "not-an-array",
		},
	}

	segments := []pathSegment{
		{Field: "spec"},
		{Field: "tablespaces", FilterKey: "name", FilterVal: "tbs1"},
		{Field: "storage"},
	}

	err := setNestedFieldWithFilter(obj, segments, "50Gi")
	if err == nil {
		t.Fatal("expected error for non-array field, got nil")
	}
	if !containsSubstring(err.Error(), "not an array") {
		t.Errorf("expected error to mention 'not an array', got: %v", err)
	}
}

func TestSetNestedFieldWithFilter_FieldMissing(t *testing.T) {
	obj := map[string]interface{}{
		"spec": map[string]interface{}{},
	}

	segments := []pathSegment{
		{Field: "spec"},
		{Field: "tablespaces", FilterKey: "name", FilterVal: "tbs1"},
		{Field: "storage"},
	}

	// Missing field should return nil (skip gracefully)
	err := setNestedFieldWithFilter(obj, segments, "50Gi")
	if err != nil {
		t.Fatalf("expected nil error for missing field, got: %v", err)
	}
}

func TestSetNestedFieldWithFilter_SimplePath(t *testing.T) {
	obj := map[string]interface{}{
		"spec": map[string]interface{}{
			"storage": map[string]interface{}{
				"size": "10Gi",
			},
		},
	}

	segments := []pathSegment{
		{Field: "spec"},
		{Field: "storage"},
		{Field: "size"},
	}

	err := setNestedFieldWithFilter(obj, segments, "50Gi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spec := obj["spec"].(map[string]interface{})
	storage := spec["storage"].(map[string]interface{})
	if storage["size"] != "50Gi" {
		t.Errorf("expected size '50Gi', got %q", storage["size"])
	}
}
