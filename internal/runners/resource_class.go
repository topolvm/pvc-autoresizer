package runners

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ResourceClass defines an admin-approved resource type and path for operator-aware resizing.
// Each class specifies which CR type and which field within that CR can be patched.
type ResourceClass struct {
	// Name is the unique identifier for this resource class (DNS label compatible)
	Name string `yaml:"name"`
	// APIGroup is the API group of the target CR (e.g., "rabbitmq.com")
	APIGroup string `yaml:"apiGroup"`
	// APIVersion is the API version of the target CR (e.g., "v1beta1")
	APIVersion string `yaml:"apiVersion"`
	// Kind is the kind of the target CR (e.g., "RabbitmqCluster")
	Kind string `yaml:"kind"`
	// Path is the JSON pointer to the storage field (must start with /spec/)
	Path string `yaml:"path"`
}

// resourceClassConfig is the top-level structure for the config file
type resourceClassConfig struct {
	ResourceClasses []ResourceClass `yaml:"resourceClasses"`
}

// GetFullAPIVersion returns the combined API version string (e.g., "rabbitmq.com/v1beta1")
func (rc *ResourceClass) GetFullAPIVersion() string {
	if rc.APIGroup == "" {
		return rc.APIVersion
	}
	return rc.APIGroup + "/" + rc.APIVersion
}

// dnsLabelRegex validates DNS label format (RFC 1123)
var dnsLabelRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// LoadResourceClasses loads and validates resource class configuration from a YAML file.
// Returns a map keyed by class name for O(1) lookup.
// Returns an empty map if the file contains no resource classes.
// Returns an error if the file cannot be read, parsed, or contains invalid configuration.
func LoadResourceClasses(path string) (map[string]ResourceClass, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read resource classes config: %w", err)
	}

	var config resourceClassConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse resource classes config: %w", err)
	}

	result := make(map[string]ResourceClass, len(config.ResourceClasses))

	for i, rc := range config.ResourceClasses {
		if err := validateResourceClass(&rc, i); err != nil {
			return nil, err
		}

		// Check for duplicate names
		if _, exists := result[rc.Name]; exists {
			return nil, fmt.Errorf("invalid resource class at index %d: duplicate name %q", i, rc.Name)
		}

		// Normalize path to JSON pointer format
		rc.Path = normalizeJSONPath(rc.Path)

		result[rc.Name] = rc
	}

	return result, nil
}

// validateResourceClass validates a single resource class configuration
func validateResourceClass(rc *ResourceClass, index int) error {
	// Validate name
	if rc.Name == "" {
		return fmt.Errorf("invalid resource class at index %d: name is required", index)
	}
	if !dnsLabelRegex.MatchString(rc.Name) {
		return fmt.Errorf("invalid resource class %q: name must be a valid DNS label (lowercase alphanumeric and hyphens, cannot start or end with hyphen)", rc.Name)
	}

	// Validate required fields
	if rc.APIGroup == "" {
		return fmt.Errorf("invalid resource class %q: apiGroup is required", rc.Name)
	}
	if rc.APIVersion == "" {
		return fmt.Errorf("invalid resource class %q: apiVersion is required", rc.Name)
	}
	if rc.Kind == "" {
		return fmt.Errorf("invalid resource class %q: kind is required", rc.Name)
	}
	if rc.Path == "" {
		return fmt.Errorf("invalid resource class %q: path is required", rc.Name)
	}

	// Validate path - normalize first, then check
	normalizedPath := normalizeJSONPath(rc.Path)
	if err := validateResourceClassPath(normalizedPath, rc.Name); err != nil {
		return err
	}

	return nil
}

// validateResourceClassPath ensures the path only targets /spec/* fields and has valid filter syntax
func validateResourceClassPath(path string, className string) error {
	// Must target a specific field under /spec (not just /spec or /spec/)
	if path == "/spec" || path == "/spec/" {
		return fmt.Errorf("invalid resource class %q: path must target a specific field under /spec (e.g., /spec/storage/size)", className)
	}

	// Must start with /spec/
	if !strings.HasPrefix(path, "/spec/") {
		return fmt.Errorf("invalid resource class %q: path %q must start with /spec/ (got %q)", className, path, path)
	}

	// Validate filter syntax if present (parsePath will catch invalid syntax)
	_, err := parsePath(path)
	if err != nil {
		return fmt.Errorf("invalid resource class %q: %w", className, err)
	}

	return nil
}
