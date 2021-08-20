package runners

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

// DefaultThreshold is the default value of ResizeThresholdAnnotation.
const DefaultThreshold = "10%"

// DefaultInodesThreshold is the default value of ResizeInodesThresholdAnnotation.
const DefaultInodesThreshold = "10%"

// DefaultIncrease is the default value of ResizeIncreaseAnnotation.
const DefaultIncrease = "10%"
