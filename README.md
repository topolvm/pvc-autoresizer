[![GitHub release](https://img.shields.io/github/v/release/topolvm/pvc-autoresizer.svg?maxAge=60)][releases]
[![Main](https://github.com/topolvm/pvc-autoresizer/workflows/Main/badge.svg)](https://github.com/topolvm/pvc-autoresizer/actions)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/topolvm/pvc-autoresizer?tab=overview)](https://pkg.go.dev/github.com/topolvm/pvc-autoresizer?tab=overview)
[![Go Report Card](https://goreportcard.com/badge/github.com/topolvm/pvc-autoresizer)](https://goreportcard.com/badge/github.com/topolvm/pvc-autoresizer)

# pvc-autoresizer

`pvc-autoresizer` is an automatic volume resizer that edits PVCs if they have less than the specified amount of free filesystem capacity.

## Target CSI Drivers

`pvc-autoresizer` supports CSI Drivers that meet the following requirements:

- implement [Volume Expansion](https://kubernetes-csi.github.io/docs/volume-expansion.html).
- implement [NodeGetVolumeStats](https://github.com/container-storage-interface/spec/blob/master/spec.md#nodegetvolumestats)

## Prepare

`pvc-autoresizer` behaves based on the metrics that prometheus collects from kubelet.

Please refer to the following pages to set up Prometheus:

- [Installation | Prometheus](https://prometheus.io/docs/prometheus/latest/installation/)

In addition, configure scraping as follows:

- [Monitoring with Prometheus | TopoLVM](https://github.com/topolvm/topolvm/blob/master/docs/prometheus.md)

## Installation

Specify the Prometheus URL to `pvc-autoresizer` argument as `--prometheus-url`.

`pvc-autoresizer` can be deployed to a Kubernetes cluster via `kustomize` and `kubectl`:

```
kustomize build ./config/default | kubectl apply -f -
```

## How to use

The StorageClass of the PVC to be resized must have `resize.topolvm.io/enabled: "true"` annotation.

```yaml
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: topolvm-provisioner
  annotations:
    resize.topolvm.io/enabled: "true" 
provisioner: topolvm.cybozu.com
parameters:
  "csi.storage.k8s.io/fstype": "xfs"
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
```

The PVC to be resized must have `.spec.resources.limits.storage` and must be `volumeMode: Filesystem`.
The PVC can have `resize.topolvm.io/threshold` and `resize.topolvm.io/increase` annotation.
(If they are not given, the default value is `10%`.)

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: topolvm-pvc
  annotations:
    resize.topolvm.io/threshold: 20%
    resize.topolvm.io/increase: 20Gi
spec:
  accessModes:
  - ReadWriteOnce
  volumeMode: Filesystem
  resources:
    requests:
      storage: 30Gi
    limits:
      storage: 100Gi
  storageClassName: topolvm-provisioner
```

When the amount of free space of the volume is smaller than `resize.topolvm.io/threshold`,
the `.spec.resources.requests.storage` size will be increased by `resize.topolvm.io/increase`.
The maximum size is specified by `.spec.resources.limits.storage`.

Container images
----------------

Container images are available on [Quay.io](https://quay.io/repository/topolvm/pvc-autoresizer)

[releases]: https://github.com/topolvm/pvc-autoresizer/releases
