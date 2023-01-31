# Expand the PVC's initial capacity based on the largest capacity in the specified PVCs.

## Summary

Expand the PVC's initial capacity based on the largest capacity in specified PVCs.

### Motivation

One use case for pvc-autoresizer is automatically expanding a PV/PVC for multiple Pod/PV pairs, such as a MySQL cluster.
In such a cluster, if a Pod/PV fails and a PV/PVC is rebuilt by the restore process, a PV is created based on the PVC template. Even if a PV of the same size as the others is actually needed, a PV of the template's capacity is created. As a result, the restore process stops due to insufficient capacity.

To solve this problem, we are planning to add a new feature to create a PV/PVC with sufficient capacity when PVCs of the same group already exist.

### Goals

- Provide the ability to increase the PVC's initial capacity based on the largest capacity in specified PVCs.
- The PVCs for which the initial capacity increase is enabled can be selected by users.
- Some application checks the PV capacity at the beginning of the process. It should be able to handle such cases.

## Proposal

### Rules for grouping & initial resize

- If a PVC being created has a `resize.topolvm.io/initial-resize-group-by` annotation and the label specified in the annotation exists, existing PVCs with matching label values are considered to be in the same group.
- If no PVCs are in the same group, or if the capacity of creating PVC is larger than any existing PVCs in the same group, the capacity of the original PVC is used as is.

Examples are given below.

```yaml
### existing PVCs (excerpted)
kind: PersistentVolumeClaim
metadata:
  name: pvc-x-1
  labels:
    label-foobar: group-x
  annotations:
    resize.topolvm.io/initial-resize-group-by: label-foobar
spec:
  resources:
    requests:
      storage: 20Gi

kind: PersistentVolumeClaim
metadata:
  name: pvc-x-2
  labels:
    label-foobar: group-x
  annotations:
    resize.topolvm.io/initial-resize-group-by: label-foobar
spec:
  resources:
    requests:
      storage: 16Gi

kind: PersistentVolumeClaim
metadata:
  name: pvc-y-1
  labels:
    label-foobar: group-y
  annotations:
    resize.topolvm.io/initial-resize-group-by: label-foobar
spec:
  resources:
    requests:
      storage: 30Gi
```

When creating the following new PVC, `pvc-x-1` and `pvc-x-2` with `label-foobar: group-x` are considered to be in the same group, and `pvc-y-1` is not. Therefore, the PVC is created with **20Gi** based on `pvc-x-1`, which has the largest capacity in the group.

```yaml
kind: PersistentVolumeClaim
metadata:
  name: pvc-x-3
  labels:
    label-foobar: group-x
  annotations:
    resize.topolvm.io/initial-resize-group-by: label-foobar
spec:
  resources:
    requests:
      storage: 10Gi
```

When creating the following new PVC, `pvc-y-1` with `label-foobar: group-y` is in the same group. However, since the new PVC's size(50Gi) is larger than the existing one(30Gi), the PVC is created with **50Gi**.

```yaml
kind: PersistentVolumeClaim
metadata:
  name: pvc-y-2
  labels:
    label-foobar: group-y
  annotations:
    resize.topolvm.io/initial-resize-group-by: label-foobar
spec:
  resources:
    requests:
      storage: 50Gi
```

### How to use

- Add `resize.topolvm.io/initial-resize-group-by` annotation to the PVC template to which you want to apply the feature.
- In most cases, annotations and labels are not added to PVCs that have already been created based on a PVC template. In such cases, it is necessary to manually add `resize.topolvm.io/initial-resize-group-by` annotation to existing PVCs in addition to the PVC template.

## Design details

### Option A) use mutating webhook for PVC

Pros:
- A restore program that checks the free capacity of the volume will not cause problems.
- We do not need to worry about expansion failure after scheduling PVs because new PVC is created with sufficient capacity.

Cons:
- In order to provide webhook features, additional configuration of mutating webhook and svc is required besides the development of the program.

### Option B) use reconcile loop for PVC

Pros:
- pvc-autoresizer already has a custom controller that monitors PVCs, so the functionality may be achieved with a few modifications.

Cons:
- A restore program that performs a free space check of the volume when using PVC may cause an error due to insufficient space before expanding PV/PVC.
- In TopoLVM, there may be cases in which PVs created cannot be expanded due to insufficient node capacity.

### Decision outcome

We adopt Option A. We have already found out that [MOCO](https://github.com/cybozu-go/moco) checks the free space of PVCs before restore process. Therefore, the restore does not work well with Option B. Other programs may behave in the same way.
