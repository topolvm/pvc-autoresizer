package pvcautoresizer

const AutoResizerAnnotationPrefix = "resize.topolvm.io/"

// AutoResizeEnabledKey is the key of flag that enables pvc-autoresizer.
const AutoResizeEnabledKey = AutoResizerAnnotationPrefix + "enabled"

// ResizeThresholdAnnotation is the key of resize threshold.
const ResizeThresholdAnnotation = AutoResizerAnnotationPrefix + "threshold"

// ResizeInodesThresholdAnnotation is the key of resize threshold for inodes.
const ResizeInodesThresholdAnnotation = AutoResizerAnnotationPrefix + "inodes-threshold"

// ResizeIncreaseAnnotation is the key of amount increased.
const ResizeIncreaseAnnotation = AutoResizerAnnotationPrefix + "increase"

// StorageLimitAnnotation is the key of storage limit value
const StorageLimitAnnotation = AutoResizerAnnotationPrefix + "storage_limit"

// PreviousCapacityBytesAnnotation is the key of previous volume capacity.
const PreviousCapacityBytesAnnotation = AutoResizerAnnotationPrefix + "pre_capacity_bytes"

// InitialResizeGroupByAnnotation is the key of the initial-resize group by.
const InitialResizeGroupByAnnotation = AutoResizerAnnotationPrefix + "initial-resize-group-by"

// AnnotationPatchingEnabled is the key of flag that enables patching of annotations for STS provisioned PVCs.
const AnnotationPatchingEnabled = AutoResizerAnnotationPrefix + "annotation-patching-enabled"

// DefaultThreshold is the default value of ResizeThresholdAnnotation.
const DefaultThreshold = "10%"

// DefaultInodesThreshold is the default value of ResizeInodesThresholdAnnotation.
const DefaultInodesThreshold = "10%"

// DefaultIncrease is the default value of ResizeIncreaseAnnotation.
const DefaultIncrease = "10%"
