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

## Architecture

![component diagram](http://www.plantuml.com/plantuml/svg/LL0zRy8m4DtzAvxIcJ_0K86OM2ea99Qgmv4SmQfZH-Spb5RzxtL88EuIw_8-thjRHINHr3dZGyDuovyV0xn_fYCxrW-yEDkUzUWI6w0XfID5nbw3KCiJUcFdmjNy6lCaK6yZouK5556jTrkCOrKOOaWIhfLywnZzfRwJTopHHcMlE0INEiR6aUsoyfapYoXf48xscsKK7pPOF_xDSQqm-qAsaz2ndZd5Si4PhwDjn3xgR_PRZEDSmXGMMAH-yOhfPi0IRNuoAhQEPjXhqOIhpvIYxaXIak79jQCfmCbna2x1Nptv1d6wUKqzrLPl__cEJveLPQibgg87mkbkeQL7DRGRm-QTvyZB_VvcnRv9dVi3)

### How pvc-autoresizer works

To expand pvc, pvc-autoresizer works as follows:

1. Get target PVC information from API server
1. Get metrics from Prometheus about storage
1. Expand PVC storage request size if PVC has less than the specified amount of free filesystem capacity. 


### Details
- Prepare yaml file as follows
```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: topolvm-pvc
  annotations:
    resize.topolvm.io/threshold: 20%
    resize.topolvm.io/amount: 20Gi
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 30Gi
    limits:
      storage: 100Gi
  storageClassName: topolvm-provisioner
```

- Annotations
To allow automatic resizing, PVC must have spec.resources.limits.storage.
TopoLVM increases PVC's spec.resources.requests.storage up to the given limits.
The threshold of free space is given by resize.topolvm.io/threshold annotation.
The amount of increased size can be specified by resize.topolvm.io/amount annotation.
The value of the annotations can be a ratio like 20% or a value like 10Gi.
The default values are 10%.
