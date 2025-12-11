package pvcautoresizer

// AutoResizeEnabledKey is the key of flag that enables pvc-autoresizer.
const AutoResizeEnabledKey = "resize.topolvm.io/enabled"

// ResizeThresholdAnnotation is the key of resize threshold.
const ResizeThresholdAnnotation = "resize.topolvm.io/threshold"

// ResizeInodesThresholdAnnotation is the key of resize threshold for inodes.
const ResizeInodesThresholdAnnotation = "resize.topolvm.io/inodes-threshold"

// ResizeIncreaseAnnotation is the key of amount increased.
const ResizeIncreaseAnnotation = "resize.topolvm.io/increase"

// StorageLimitAnnotation is the key of storage limit value
const StorageLimitAnnotation = "resize.topolvm.io/storage_limit"

// PreviousCapacityBytesAnnotation is the key of previous volume capacity.
const PreviousCapacityBytesAnnotation = "resize.topolvm.io/pre_capacity_bytes"

// InitialResizeGroupByAnnotation is the key of the initial-resize group by.
const InitialResizeGroupByAnnotation = "resize.topolvm.io/initial-resize-group-by"

// TargetResourceAPIVersionAnnotation is the API version of the target Custom Resource to patch.
const TargetResourceAPIVersionAnnotation = "resize.topolvm.io/target-resource-api-version"

// TargetResourceKindAnnotation is the Kind of the target Custom Resource to patch.
const TargetResourceKindAnnotation = "resize.topolvm.io/target-resource-kind"

// TargetResourceNameAnnotation is the name of the target Custom Resource to patch.
const TargetResourceNameAnnotation = "resize.topolvm.io/target-resource-name"

// TargetResourceNamespaceAnnotation is the namespace of the target Custom Resource (defaults to PVC namespace).
const TargetResourceNamespaceAnnotation = "resize.topolvm.io/target-resource-namespace"

// TargetResourceJSONPathAnnotation is the JSON path to the storage field in the target Custom Resource.
const TargetResourceJSONPathAnnotation = "resize.topolvm.io/target-resource-json-path"

// DefaultThreshold is the default value of ResizeThresholdAnnotation.
const DefaultThreshold = "10%"

// DefaultInodesThreshold is the default value of ResizeInodesThresholdAnnotation.
const DefaultInodesThreshold = "10%"

// DefaultIncrease is the default value of ResizeIncreaseAnnotation.
const DefaultIncrease = "10%"
