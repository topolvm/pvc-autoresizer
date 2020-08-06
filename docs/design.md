# Design Notes


## Motivation

Persistent Volume(PV) is used for persistenting data (ex. MySql, Elasticsearch).
These data size will increase in the future.
It is difficult to estimate these in advance.
So, PV should be automatically expanded based on the PV usage.

## Goal

- Automatic resizing PVC
- User can set parameter for auto resizing using PVC annotations
- The target storage class to be resized can be specified using its annotations

## Target

- CSI plugins which supports [`VolumeExpansion`](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#csi-volume-expansion)
- Only FileSystem volume mode (not Raw)





What
Implement an automatic volume resizer in TopoLVM that edits PVCs if they have less than the specified amount of free filesystem capacity.
How
To allow automatic resizing, PVC must have spec.resources.limits.storage.
TopoLVM increases PVC's spec.resources.requests.storage up to the given limits.
The threshold of free space is given by resize.topolvm.io/threshold annotation.
The amount of increased size can be specified by resize.topolvm.io/amount annotation.
The value of the annotations can be a ratio like 20% or a value like 10Gi.
The default values are 10%.
