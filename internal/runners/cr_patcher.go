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
// for the CR types listed in operatorAwareResizing.allowedResources.
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

// parseCRTargetAnnotations extracts and validates CR target annotations from a PVC.
// Returns nil if no CR target annotations are present.
// Returns error if annotations are present but invalid or incomplete.
func parseCRTargetAnnotations(pvc *corev1.PersistentVolumeClaim) (*CRTargetConfig, error) {
	annotations := pvc.Annotations
	if annotations == nil {
		return nil, nil
	}

	// Check if any target-resource annotation is present
	apiVersion := annotations[pvcautoresizer.TargetResourceAPIVersionAnnotation]
	kind := annotations[pvcautoresizer.TargetResourceKindAnnotation]
	name := annotations[pvcautoresizer.TargetResourceNameAnnotation]
	namespace := annotations[pvcautoresizer.TargetResourceNamespaceAnnotation]
	jsonPath := annotations[pvcautoresizer.TargetResourceJSONPathAnnotation]

	// If none of the annotations are present, return nil (not configured for CR patching)
	if apiVersion == "" && kind == "" && name == "" && jsonPath == "" {
		return nil, nil
	}

	// All-or-nothing validation: if any annotation is present, validate all required ones
	var missingFields []string
	if apiVersion == "" {
		missingFields = append(missingFields, "api-version")
	}
	if kind == "" {
		missingFields = append(missingFields, "kind")
	}
	if name == "" {
		missingFields = append(missingFields, "name")
	}
	if jsonPath == "" {
		missingFields = append(missingFields, "json-path")
	}

	if len(missingFields) > 0 {
		return nil, fmt.Errorf("incomplete CR target configuration: missing required annotations: %s",
			strings.Join(missingFields, ", "))
	}

	// Default namespace to PVC namespace if not specified
	if namespace == "" {
		namespace = pvc.Namespace
	}

	// Normalize JSON path to JSON Pointer format
	normalizedPath := normalizeJSONPath(jsonPath)

	// Validate path security - only allow /spec/* paths
	if err := validateJSONPath(jsonPath); err != nil {
		return nil, err
	}

	return &CRTargetConfig{
		APIVersion: apiVersion,
		Kind:       kind,
		Name:       name,
		Namespace:  namespace,
		JSONPath:   normalizedPath,
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

	// Update the field value at the JSON path
	// Split the path into segments (remove leading slash and split by /)
	pathSegments := strings.Split(strings.TrimPrefix(target.JSONPath, "/"), "/")
	if len(pathSegments) == 0 || (len(pathSegments) == 1 && pathSegments[0] == "") {
		return fmt.Errorf("invalid JSON path: %q", target.JSONPath)
	}

	// Use SetNestedField to update the value
	// Convert the Quantity to a string for storage in the CR
	err = unstructured.SetNestedField(obj.Object, newSize.String(), pathSegments...)
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
