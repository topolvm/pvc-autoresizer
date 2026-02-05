package runners

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadResourceClasses_ValidSingleClass(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: "rabbitmq"
    apiGroup: "rabbitmq.com"
    apiVersion: "v1beta1"
    kind: "RabbitmqCluster"
    path: "/spec/persistence/storage"
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	classes, err := LoadResourceClasses(configPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(classes) != 1 {
		t.Errorf("expected 1 class, got %d", len(classes))
	}
	if _, ok := classes["rabbitmq"]; !ok {
		t.Error("expected 'rabbitmq' class to exist")
	}

	rc := classes["rabbitmq"]
	if rc.Name != "rabbitmq" {
		t.Errorf("expected name 'rabbitmq', got %q", rc.Name)
	}
	if rc.APIGroup != "rabbitmq.com" {
		t.Errorf("expected apiGroup 'rabbitmq.com', got %q", rc.APIGroup)
	}
	if rc.APIVersion != "v1beta1" {
		t.Errorf("expected apiVersion 'v1beta1', got %q", rc.APIVersion)
	}
	if rc.Kind != "RabbitmqCluster" {
		t.Errorf("expected kind 'RabbitmqCluster', got %q", rc.Kind)
	}
	if rc.Path != "/spec/persistence/storage" {
		t.Errorf("expected path '/spec/persistence/storage', got %q", rc.Path)
	}
}

func TestLoadResourceClasses_ValidMultipleClasses(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: "rabbitmq"
    apiGroup: "rabbitmq.com"
    apiVersion: "v1beta1"
    kind: "RabbitmqCluster"
    path: "/spec/persistence/storage"
  - name: "cnpg-data"
    apiGroup: "postgresql.cnpg.io"
    apiVersion: "v1"
    kind: "Cluster"
    path: "/spec/storage/size"
  - name: "cnpg-wal"
    apiGroup: "postgresql.cnpg.io"
    apiVersion: "v1"
    kind: "Cluster"
    path: "/spec/walStorage/size"
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	classes, err := LoadResourceClasses(configPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(classes) != 3 {
		t.Errorf("expected 3 classes, got %d", len(classes))
	}
	if _, ok := classes["rabbitmq"]; !ok {
		t.Error("expected 'rabbitmq' class to exist")
	}
	if _, ok := classes["cnpg-data"]; !ok {
		t.Error("expected 'cnpg-data' class to exist")
	}
	if _, ok := classes["cnpg-wal"]; !ok {
		t.Error("expected 'cnpg-wal' class to exist")
	}

	// Verify different paths for same CR type
	if classes["cnpg-data"].Path != "/spec/storage/size" {
		t.Errorf("expected cnpg-data path '/spec/storage/size', got %q", classes["cnpg-data"].Path)
	}
	if classes["cnpg-wal"].Path != "/spec/walStorage/size" {
		t.Errorf("expected cnpg-wal path '/spec/walStorage/size', got %q", classes["cnpg-wal"].Path)
	}
}

func TestLoadResourceClasses_EmptyList(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`resourceClasses: []`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	classes, err := LoadResourceClasses(configPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(classes) != 0 {
		t.Errorf("expected empty map, got %d classes", len(classes))
	}
}

func TestLoadResourceClasses_MissingKey(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`# Empty config`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	classes, err := LoadResourceClasses(configPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(classes) != 0 {
		t.Errorf("expected empty map, got %d classes", len(classes))
	}
}

func TestLoadResourceClasses_NormalizeDotNotation(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: "test"
    apiGroup: "example.com"
    apiVersion: "v1"
    kind: "Example"
    path: ".spec.storage.size"
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	classes, err := LoadResourceClasses(configPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if classes["test"].Path != "/spec/storage/size" {
		t.Errorf("expected normalized path '/spec/storage/size', got %q", classes["test"].Path)
	}
}

func TestLoadResourceClasses_NormalizeWithoutLeadingDot(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: "test"
    apiGroup: "example.com"
    apiVersion: "v1"
    kind: "Example"
    path: "spec.storage.size"
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	classes, err := LoadResourceClasses(configPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if classes["test"].Path != "/spec/storage/size" {
		t.Errorf("expected normalized path '/spec/storage/size', got %q", classes["test"].Path)
	}
}

func TestLoadResourceClasses_RejectsDuplicateNames(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: "rabbitmq"
    apiGroup: "rabbitmq.com"
    apiVersion: "v1beta1"
    kind: "RabbitmqCluster"
    path: "/spec/persistence/storage"
  - name: "rabbitmq"
    apiGroup: "rabbitmq.com"
    apiVersion: "v1beta1"
    kind: "RabbitmqCluster"
    path: "/spec/other/field"
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = LoadResourceClasses(configPath)
	if err == nil {
		t.Fatal("expected error for duplicate names, got nil")
	}
	if !contains(err.Error(), "duplicate") || !contains(err.Error(), "rabbitmq") {
		t.Errorf("expected error to mention 'duplicate' and 'rabbitmq', got: %v", err)
	}
}

func TestLoadResourceClasses_RejectsInvalidPath_Metadata(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: "bad-path"
    apiGroup: "example.com"
    apiVersion: "v1"
    kind: "Example"
    path: "/metadata/name"
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = LoadResourceClasses(configPath)
	if err == nil {
		t.Fatal("expected error for /metadata path, got nil")
	}
	if !contains(err.Error(), "/spec/") {
		t.Errorf("expected error to mention '/spec/', got: %v", err)
	}
}

func TestLoadResourceClasses_RejectsInvalidPath_Status(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: "bad-status"
    apiGroup: "example.com"
    apiVersion: "v1"
    kind: "Example"
    path: "/status/conditions"
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = LoadResourceClasses(configPath)
	if err == nil {
		t.Fatal("expected error for /status path, got nil")
	}
	if !contains(err.Error(), "/spec/") {
		t.Errorf("expected error to mention '/spec/', got: %v", err)
	}
}

func TestLoadResourceClasses_RejectsEmptyName(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: ""
    apiGroup: "example.com"
    apiVersion: "v1"
    kind: "Example"
    path: "/spec/storage/size"
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = LoadResourceClasses(configPath)
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if !contains(err.Error(), "name") || !contains(err.Error(), "required") {
		t.Errorf("expected error to mention 'name' and 'required', got: %v", err)
	}
}

func TestLoadResourceClasses_RejectsMissingAPIGroup(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: "test"
    apiVersion: "v1"
    kind: "Example"
    path: "/spec/storage/size"
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = LoadResourceClasses(configPath)
	if err == nil {
		t.Fatal("expected error for missing apiGroup, got nil")
	}
	if !contains(err.Error(), "apiGroup") || !contains(err.Error(), "required") {
		t.Errorf("expected error to mention 'apiGroup' and 'required', got: %v", err)
	}
}

func TestLoadResourceClasses_RejectsMissingAPIVersion(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: "test"
    apiGroup: "example.com"
    kind: "Example"
    path: "/spec/storage/size"
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = LoadResourceClasses(configPath)
	if err == nil {
		t.Fatal("expected error for missing apiVersion, got nil")
	}
	if !contains(err.Error(), "apiVersion") || !contains(err.Error(), "required") {
		t.Errorf("expected error to mention 'apiVersion' and 'required', got: %v", err)
	}
}

func TestLoadResourceClasses_RejectsMissingKind(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: "test"
    apiGroup: "example.com"
    apiVersion: "v1"
    path: "/spec/storage/size"
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = LoadResourceClasses(configPath)
	if err == nil {
		t.Fatal("expected error for missing kind, got nil")
	}
	if !contains(err.Error(), "kind") || !contains(err.Error(), "required") {
		t.Errorf("expected error to mention 'kind' and 'required', got: %v", err)
	}
}

func TestLoadResourceClasses_RejectsMissingPath(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: "test"
    apiGroup: "example.com"
    apiVersion: "v1"
    kind: "Example"
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = LoadResourceClasses(configPath)
	if err == nil {
		t.Fatal("expected error for missing path, got nil")
	}
	if !contains(err.Error(), "path") || !contains(err.Error(), "required") {
		t.Errorf("expected error to mention 'path' and 'required', got: %v", err)
	}
}

func TestLoadResourceClasses_RejectsPathJustSpec(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: "test"
    apiGroup: "example.com"
    apiVersion: "v1"
    kind: "Example"
    path: "/spec"
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = LoadResourceClasses(configPath)
	if err == nil {
		t.Fatal("expected error for path '/spec' without specific field, got nil")
	}
	if !contains(err.Error(), "specific field") {
		t.Errorf("expected error to mention 'specific field', got: %v", err)
	}
}

func TestLoadResourceClasses_RejectsInvalidDNSLabel_Underscore(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: "Invalid_Name"
    apiGroup: "example.com"
    apiVersion: "v1"
    kind: "Example"
    path: "/spec/storage/size"
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = LoadResourceClasses(configPath)
	if err == nil {
		t.Fatal("expected error for invalid DNS label name, got nil")
	}
	if !contains(err.Error(), "DNS label") {
		t.Errorf("expected error to mention 'DNS label', got: %v", err)
	}
}

func TestLoadResourceClasses_RejectsInvalidDNSLabel_StartingHyphen(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: "-invalid"
    apiGroup: "example.com"
    apiVersion: "v1"
    kind: "Example"
    path: "/spec/storage/size"
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = LoadResourceClasses(configPath)
	if err == nil {
		t.Fatal("expected error for name starting with hyphen, got nil")
	}
	if !contains(err.Error(), "DNS label") {
		t.Errorf("expected error to mention 'DNS label', got: %v", err)
	}
}

func TestLoadResourceClasses_NonExistentFile(t *testing.T) {
	_, err := LoadResourceClasses("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestLoadResourceClasses_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: "test"
    invalid yaml here: [[[
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = LoadResourceClasses(configPath)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestResourceClass_GetFullAPIVersion_WithGroup(t *testing.T) {
	rc := ResourceClass{
		APIGroup:   "rabbitmq.com",
		APIVersion: "v1beta1",
	}
	expected := "rabbitmq.com/v1beta1"
	if got := rc.GetFullAPIVersion(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestResourceClass_GetFullAPIVersion_CoreGroup(t *testing.T) {
	rc := ResourceClass{
		APIGroup:   "",
		APIVersion: "v1",
	}
	expected := "v1"
	if got := rc.GetFullAPIVersion(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestLoadResourceClasses_ValidPathWithFilter(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: "cnpg-tablespace"
    apiGroup: "postgresql.cnpg.io"
    apiVersion: "v1"
    kind: "Cluster"
    path: "/spec/tablespaces[name=?]/storage/size"
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	classes, err := LoadResourceClasses(configPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(classes) != 1 {
		t.Errorf("expected 1 class, got %d", len(classes))
	}
	if classes["cnpg-tablespace"].Path != "/spec/tablespaces[name=?]/storage/size" {
		t.Errorf("expected path '/spec/tablespaces[name=?]/storage/size', got %q", classes["cnpg-tablespace"].Path)
	}
}

func TestLoadResourceClasses_ValidPathWithHardcodedFilter(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: "cnpg-tbs1"
    apiGroup: "postgresql.cnpg.io"
    apiVersion: "v1"
    kind: "Cluster"
    path: "/spec/tablespaces[name=tbs1]/storage/size"
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	classes, err := LoadResourceClasses(configPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if classes["cnpg-tbs1"].Path != "/spec/tablespaces[name=tbs1]/storage/size" {
		t.Errorf("expected path with hardcoded filter, got %q", classes["cnpg-tbs1"].Path)
	}
}

func TestLoadResourceClasses_RejectsInvalidFilterSyntax(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: "bad-filter"
    apiGroup: "postgresql.cnpg.io"
    apiVersion: "v1"
    kind: "Cluster"
    path: "/spec/tablespaces[name/storage/size"
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = LoadResourceClasses(configPath)
	if err == nil {
		t.Fatal("expected error for invalid filter syntax, got nil")
	}
	if !contains(err.Error(), "filter") || !contains(err.Error(), "bracket") {
		t.Errorf("expected error to mention 'filter' and 'bracket', got: %v", err)
	}
}

func TestLoadResourceClasses_RejectsMultipleFilters(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "resource-classes.yaml")
	err := os.WriteFile(configPath, []byte(`
resourceClasses:
  - name: "multi-filter"
    apiGroup: "postgresql.cnpg.io"
    apiVersion: "v1"
    kind: "Cluster"
    path: "/spec/tablespaces[name=?]/volumes[id=?]/size"
`), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = LoadResourceClasses(configPath)
	if err == nil {
		t.Fatal("expected error for multiple filters, got nil")
	}
	if !contains(err.Error(), "multiple") {
		t.Errorf("expected error to mention 'multiple', got: %v", err)
	}
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
