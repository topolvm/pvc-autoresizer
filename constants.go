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

// TargetResourceClassAnnotation is the name of the admin-defined resource class for operator-aware resizing.
// When set, the controller looks up the resource class from the admin configuration and patches the
// corresponding CR field instead of the PVC directly.
const TargetResourceClassAnnotation = "resize.topolvm.io/target-resource-class"

// TargetResourceNameAnnotation is the name of the target Custom Resource to patch.
// Required when using TargetResourceClassAnnotation. The CR must be in the same namespace as the PVC.
const TargetResourceNameAnnotation = "resize.topolvm.io/target-resource-name"

// TargetFilterValueAnnotation specifies the value for array element selection in paths with [key=?] syntax.
// Required when the resource class path contains a placeholder filter like [name=?].
const TargetFilterValueAnnotation = "resize.topolvm.io/target-filter-value"

// DefaultThreshold is the default value of ResizeThresholdAnnotation.
const DefaultThreshold = "10%"

// DefaultInodesThreshold is the default value of ResizeInodesThresholdAnnotation.
const DefaultInodesThreshold = "10%"

// DefaultIncrease is the default value of ResizeIncreaseAnnotation.
const DefaultIncrease = "10%"
