package runners

import (
	"context"
	"fmt"
	"strings"

	pvcautoresizer "github.com/topolvm/pvc-autoresizer"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RBAC for operator-aware resizing is configured via Helm values (values.yaml).
// When operatorAwareResizing.enabled is true, RBAC rules are automatically generated
// for the CR types listed in operatorAwareResizing.resourceClasses.
// See docs/operator-aware-resizing.md for configuration details.

// CRTargetConfig holds the parsed Custom Resource target configuration
// extracted from PVC annotations.
type CRTargetConfig struct {
	// APIVersion is the API version of the target CR (e.g., "rabbitmq.com/v1beta1")
	APIVersion string
	// Kind is the kind of the target CR (e.g., "RabbitmqCluster")
	Kind string
	// Name is the name of the target CR instance
	Name string
	// Namespace is the namespace of the target CR (defaults to PVC namespace if not specified)
	Namespace string
	// JSONPath is the JSON path to the storage field in the CR (normalized to JSON Pointer format)
	JSONPath string
}

// pathSegment represents a single segment in a JSON path, optionally with a filter.
// Example: "tablespaces[name=tbs1]" becomes {Field: "tablespaces", FilterKey: "name", FilterVal: "tbs1"}
type pathSegment struct {
	Field     string // The field name (e.g., "tablespaces")
	FilterKey string // The filter key (e.g., "name"), empty if no filter
	FilterVal string // The filter value (e.g., "tbs1" or "?" for placeholder), empty if no filter
}

// parsePath parses a JSON Pointer path into segments, extracting any filters.
// Supports paths like:
//   - "/spec/storage/size" (simple path)
//   - "/spec/tablespaces[name=tbs1]/storage/size" (path with filter)
//   - "/spec/tablespaces[name=?]/storage/size" (path with placeholder)
//
// Only one filter per path is allowed.
func parsePath(path string) ([]pathSegment, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	// Remove leading slash
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return nil, fmt.Errorf("path cannot be just '/'")
	}

	// Split by slash
	parts := strings.Split(path, "/")
	segments := make([]pathSegment, 0, len(parts))
	filterCount := 0

	for _, part := range parts {
		if part == "" {
			continue
		}

		segment, hasFilter, err := parsePathSegment(part)
		if err != nil {
			return nil, err
		}

		if hasFilter {
			filterCount++
			if filterCount > 1 {
				return nil, fmt.Errorf("path contains multiple filters; only one filter is allowed")
			}
		}

		segments = append(segments, segment)
	}

	if len(segments) == 0 {
		return nil, fmt.Errorf("path has no valid segments")
	}

	return segments, nil
}

// parsePathSegment parses a single path segment like "tablespaces[name=tbs1]" or "storage".
// Returns the parsed segment, whether it has a filter, and any error.
func parsePathSegment(part string) (pathSegment, bool, error) {
	// Check for filter syntax: field[key=value]
	bracketStart := strings.Index(part, "[")
	if bracketStart == -1 {
		// No filter
		return pathSegment{Field: part}, false, nil
	}

	// Has a filter - validate syntax
	bracketEnd := strings.Index(part, "]")
	if bracketEnd == -1 {
		return pathSegment{}, false, fmt.Errorf("invalid filter syntax in %q: missing closing bracket", part)
	}
	if bracketEnd != len(part)-1 {
		return pathSegment{}, false, fmt.Errorf("invalid filter syntax in %q: characters after closing bracket", part)
	}

	field := part[:bracketStart]
	if field == "" {
		return pathSegment{}, false, fmt.Errorf("invalid filter syntax in %q: empty field name", part)
	}

	filterExpr := part[bracketStart+1 : bracketEnd]
	if filterExpr == "" {
		return pathSegment{}, false, fmt.Errorf("invalid filter syntax in %q: empty filter expression", part)
	}

	// Parse key=value
	eqIndex := strings.Index(filterExpr, "=")
	if eqIndex == -1 {
		return pathSegment{}, false, fmt.Errorf("invalid filter syntax in %q: missing '=' in filter", part)
	}

	filterKey := filterExpr[:eqIndex]
	filterVal := filterExpr[eqIndex+1:]

	if filterKey == "" {
		return pathSegment{}, false, fmt.Errorf("invalid filter syntax in %q: empty filter key", part)
	}
	// Note: filterVal can be empty string or "?" for placeholder

	return pathSegment{
		Field:     field,
		FilterKey: filterKey,
		FilterVal: filterVal,
	}, true, nil
}

// resolvePlaceholder resolves any "?" placeholder in the path segments with the given filter value.
// Returns a copy of the segments with the placeholder replaced.
// Returns error if a placeholder exists but filterValue is empty.
func resolvePlaceholder(segments []pathSegment, filterValue string) ([]pathSegment, error) {
	// Check if any segment has a placeholder
	hasPlaceholder := false
	for _, seg := range segments {
		if seg.FilterVal == "?" {
			hasPlaceholder = true
			break
		}
	}

	// If there's a placeholder, filterValue is required
	if hasPlaceholder && filterValue == "" {
		return nil, fmt.Errorf("path contains placeholder [key=?] but target-filter-value annotation is missing or empty")
	}

	// Create a copy with placeholder resolved
	resolved := make([]pathSegment, len(segments))
	for i, seg := range segments {
		resolved[i] = seg
		if seg.FilterVal == "?" {
			resolved[i].FilterVal = filterValue
		}
	}

	return resolved, nil
}

// setNestedFieldWithFilter navigates through an object using path segments and sets the final value.
// Supports array filtering with [key=value] syntax.
// Returns nil if an array element is not found (graceful skip).
// Returns error if field is not an array when filter is specified, or multiple elements match.
func setNestedFieldWithFilter(obj map[string]interface{}, segments []pathSegment, value interface{}) error {
	if len(segments) == 0 {
		return fmt.Errorf("path has no segments")
	}

	// Navigate to the parent of the final field
	current := obj
	for i := 0; i < len(segments)-1; i++ {
		seg := segments[i]

		val, exists := current[seg.Field]
		if !exists {
			// Field doesn't exist - skip gracefully
			return nil
		}

		if seg.FilterKey != "" {
			// This segment has a filter - val must be an array
			arr, ok := val.([]interface{})
			if !ok {
				return fmt.Errorf("field %q is not an array (filter [%s=%s] requires array)",
					seg.Field, seg.FilterKey, seg.FilterVal)
			}

			// Find the matching element
			match, err := findArrayElement(arr, seg.FilterKey, seg.FilterVal)
			if err != nil {
				return fmt.Errorf("error finding element in %q: %w", seg.Field, err)
			}
			if match == nil {
				// No match found - skip gracefully
				return nil
			}

			current = match
		} else {
			// No filter - val must be a map
			next, ok := val.(map[string]interface{})
			if !ok {
				return fmt.Errorf("field %q is not an object", seg.Field)
			}
			current = next
		}
	}

	// Set the final field
	finalSeg := segments[len(segments)-1]
	if finalSeg.FilterKey != "" {
		return fmt.Errorf("filter not allowed on final path segment")
	}
	current[finalSeg.Field] = value

	return nil
}

// findArrayElement finds a single element in an array where element[key] == value.
// Returns nil if no element matches.
// Returns error if multiple elements match.
func findArrayElement(arr []interface{}, key, value string) (map[string]interface{}, error) {
	var match map[string]interface{}
	matchCount := 0

	for _, item := range arr {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		fieldVal, exists := obj[key]
		if !exists {
			continue
		}

		// Compare as string
		fieldStr, ok := fieldVal.(string)
		if !ok {
			continue
		}

		if fieldStr == value {
			match = obj
			matchCount++
		}
	}

	if matchCount > 1 {
		return nil, fmt.Errorf("multiple elements match [%s=%s] (expected exactly one)", key, value)
	}

	return match, nil
}

// parseCRTargetAnnotations extracts CR target configuration from PVC annotations
// using admin-defined resource classes.
// Returns nil if no CR target annotations are present.
// Returns error if annotations are present but invalid or the class is not defined.
func parseCRTargetAnnotations(pvc *corev1.PersistentVolumeClaim, resourceClasses map[string]ResourceClass) (*CRTargetConfig, error) {
	annotations := pvc.Annotations
	if annotations == nil {
		return nil, nil
	}

	// Check if the class-based annotation is present
	className := annotations[pvcautoresizer.TargetResourceClassAnnotation]
	if className == "" {
		return nil, nil
	}

	// Validate that resource classes are configured
	if len(resourceClasses) == 0 {
		return nil, fmt.Errorf("resource classes not configured: cannot use target-resource-class annotation")
	}

	// Look up the resource class
	resourceClass, exists := resourceClasses[className]
	if !exists {
		return nil, fmt.Errorf("unknown resource class %q: not defined in controller configuration", className)
	}

	// Validate that the name annotation is present
	name := annotations[pvcautoresizer.TargetResourceNameAnnotation]
	if name == "" {
		return nil, fmt.Errorf("missing required annotation: target-resource-name (required when using target-resource-class)")
	}

	// CR must be in the same namespace as the PVC (cross-namespace not supported)
	namespace := pvc.Namespace

	return &CRTargetConfig{
		APIVersion: resourceClass.GetFullAPIVersion(),
		Kind:       resourceClass.Kind,
		Name:       name,
		Namespace:  namespace,
		JSONPath:   resourceClass.Path,
	}, nil
}

// normalizeJSONPath converts dot notation to JSON Pointer (RFC 6901) format.
// Examples:
//   - ".spec.persistence.storage" → "/spec/persistence/storage"
//   - "spec.persistence.storage" → "/spec/persistence/storage"
//   - "/spec/persistence/storage" → "/spec/persistence/storage" (already normalized)
func normalizeJSONPath(path string) string {
	// Remove leading dot if present
	path = strings.TrimPrefix(path, ".")

	// If already in JSON Pointer format (starts with /), return as-is
	if strings.HasPrefix(path, "/") {
		return path
	}

	// Convert dot notation to JSON Pointer format
	// Replace dots with slashes and add leading slash
	return "/" + strings.ReplaceAll(path, ".", "/")
}

// validateJSONPath ensures the JSON path only targets /spec/* fields.
// This prevents privilege escalation by blocking patches to:
// - /metadata (labels, annotations, ownerRefs, etc.)
// - /status (operator state)
// - Other sensitive fields
func validateJSONPath(path string) error {
	// Normalize to JSON Pointer format first
	normalized := normalizeJSONPath(path)

	// Must start with /spec/
	if !strings.HasPrefix(normalized, "/spec/") {
		return fmt.Errorf("invalid JSON path %q: for security reasons, only paths starting with /spec/ are allowed (got %q)",
			path, normalized)
	}

	// Additional validation: ensure it's not just "/spec" (must target a field under spec)
	if normalized == "/spec" || normalized == "/spec/" {
		return fmt.Errorf("invalid JSON path %q: must target a specific field under /spec (e.g., /spec/storage/size)",
			path)
	}

	return nil
}

// patchCRField patches the target Custom Resource field with the new storage size.
// This method is called when a PVC has CR target annotations configured.
func (w *pvcAutoresizer) patchCRField(ctx context.Context, pvc *corev1.PersistentVolumeClaim,
	target *CRTargetConfig, newSize *resource.Quantity) error {

	log := w.log.WithName("patchCRField").WithValues(
		"pvc_namespace", pvc.Namespace,
		"pvc_name", pvc.Name,
		"cr_kind", target.Kind,
		"cr_namespace", target.Namespace,
		"cr_name", target.Name,
		"cr_path", target.JSONPath,
	)

	// Parse the APIVersion into group and version
	group, version, err := parseAPIVersion(target.APIVersion)
	if err != nil {
		return fmt.Errorf("invalid API version %q: %w", target.APIVersion, err)
	}

	// Create an unstructured object with the correct GVK
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    target.Kind,
	})

	// Get the current CR
	key := types.NamespacedName{
		Namespace: target.Namespace,
		Name:      target.Name,
	}

	err = w.client.Get(ctx, key, obj)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("target CR %s/%s/%s not found", target.Kind, target.Namespace, target.Name)
		}
		if apierrors.IsForbidden(err) {
			return fmt.Errorf("insufficient permissions to get CR %s/%s/%s: %w. "+
				"Add RBAC rule: apiGroups: [\"%s\"], resources: [\"%s\"], verbs: [\"get\", \"list\", \"patch\"]",
				target.Kind, target.Namespace, target.Name, err, group, strings.ToLower(target.Kind)+"s")
		}
		return fmt.Errorf("failed to get CR %s/%s/%s: %w", target.Kind, target.Namespace, target.Name, err)
	}

	// Store the original for creating a patch
	original := obj.DeepCopy()

	// Parse the path with filter support
	segments, err := parsePath(target.JSONPath)
	if err != nil {
		return fmt.Errorf("invalid JSON path %q: %w", target.JSONPath, err)
	}

	// Get filter value annotation and resolve placeholders
	filterValue := pvc.Annotations[pvcautoresizer.TargetFilterValueAnnotation]
	resolvedSegments, err := resolvePlaceholder(segments, filterValue)
	if err != nil {
		return fmt.Errorf("failed to resolve path placeholder: %w", err)
	}

	// Update the field value at the JSON path (with filter support)
	err = setNestedFieldWithFilter(obj.Object, resolvedSegments, newSize.String())
	if err != nil {
		return fmt.Errorf("failed to set field at path %q: %w", target.JSONPath, err)
	}

	// Create and apply the patch
	patch := client.MergeFrom(original)
	err = w.client.Patch(ctx, obj, patch)
	if err != nil {
		if apierrors.IsForbidden(err) {
			return fmt.Errorf("insufficient permissions to patch CR %s/%s/%s: %w. "+
				"Add RBAC rule: apiGroups: [\"%s\"], resources: [\"%s\"], verbs: [\"get\", \"list\", \"patch\"]",
				target.Kind, target.Namespace, target.Name, err, group, strings.ToLower(target.Kind)+"s")
		}
		if apierrors.IsConflict(err) {
			return fmt.Errorf("conflict while patching CR %s/%s/%s (will retry): %w",
				target.Kind, target.Namespace, target.Name, err)
		}
		return fmt.Errorf("failed to patch CR %s/%s/%s: %w", target.Kind, target.Namespace, target.Name, err)
	}

	log.Info("successfully patched CR field", "new_size", newSize.String())
	return nil
}

// parseAPIVersion splits an APIVersion string into group and version.
// Examples:
//   - "rabbitmq.com/v1beta1" → ("rabbitmq.com", "v1beta1")
//   - "v1" → ("", "v1")  // core API group
func parseAPIVersion(apiVersion string) (group string, version string, err error) {
	if apiVersion == "" {
		return "", "", fmt.Errorf("API version cannot be empty")
	}

	parts := strings.SplitN(apiVersion, "/", 2)
	if len(parts) == 1 {
		// Core API group (e.g., "v1")
		return "", parts[0], nil
	}

	// Custom resource group (e.g., "rabbitmq.com/v1beta1")
	return parts[0], parts[1], nil
}
