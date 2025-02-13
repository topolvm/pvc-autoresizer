[![GitHub release](https://img.shields.io/github/v/release/topolvm/pvc-autoresizer.svg?maxAge=60)][releases]
[![Main](https://github.com/topolvm/pvc-autoresizer/workflows/Main/badge.svg)](https://github.com/topolvm/pvc-autoresizer/actions)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/topolvm/pvc-autoresizer?tab=overview)](https://pkg.go.dev/github.com/topolvm/pvc-autoresizer?tab=overview)
[![Go Report Card](https://goreportcard.com/badge/github.com/topolvm/pvc-autoresizer)](https://goreportcard.com/badge/github.com/topolvm/pvc-autoresizer)

# Welcome to the pvc-autoresizer Project!

`pvc-autoresizer` resizes PersistentVolumeClaims (PVCs) when the free amount of storage is below the threshold.

It queries the volume usage metrics from Prometheus that collects metrics from `kubelet`.

Our supported platforms are:

- Kubernetes: 1.32, 1.31, 1.30
- CSI drivers that implements the following features
  - [Volume Expansion](https://kubernetes-csi.github.io/docs/volume-expansion.html)
  - [NodeGetVolumeStats](https://github.com/container-storage-interface/spec/blob/master/spec.md#nodegetvolumestats)
- CPU Architecture: x86 (\*1), arm64 (\*2)

\*1 Tier1 support. The official docker images are provided and all functionalities are tested by CI.  
\*2 Tier2 support. The official docker images are provided, but no tests run by CI.

Container images are available on [ghcr.io](https://github.com/topolvm/pvc-autoresizer/pkgs/container/pvc-autoresizer).  

## Getting Started

### Prepare

`pvc-autoresizer` behaves based on the metrics that prometheus collects from kubelet.

Please refer to the following pages to set up Prometheus:

- [Installation | Prometheus](https://prometheus.io/docs/prometheus/latest/installation/)

In addition, configure scraping as follows:

- [Monitoring with Prometheus | TopoLVM](https://github.com/topolvm/topolvm/blob/master/docs/prometheus.md)

### Installation

Specify the Prometheus URL to `pvc-autoresizer` argument as `--prometheus-url`.

`pvc-autoresizer` can be deployed to a Kubernetes cluster via `helm`:

```sh
helm repo add pvc-autoresizer https://topolvm.github.io/pvc-autoresizer/
helm install --create-namespace --namespace pvc-autoresizer pvc-autoresizer pvc-autoresizer/pvc-autoresizer --set "controller.args.prometheusURL=<YOUR PROMETHEUS ENDPOINT>"
```

See the Chart [README.md](./charts/pvc-autoresizer/README.md) for detailed documentation on the Helm Chart.

### How to use

To allow auto volume expansion, the StorageClass of PVC need to allow volume expansion and
have `resize.topolvm.io/enabled: "true"` annotation.  The annotation may be omitted if
you give `--no-annotation-check` command-line flag to `pvc-autoresizer` executable.

```yaml
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: topolvm-provisioner
  annotations:
    resize.topolvm.io/enabled: "true"
provisioner: topolvm.io
allowVolumeExpansion: true
```

To allow auto volume expansion, the PVC to be resized needs to specify the upper limit of
volume size with the annotation `resize.topolvm.io/storage_limit`.
The value of `resize.topolvm.io/storage_limit` should not be zero,
or the annotation will be ignored.

The PVC must have `volumeMode: Filesystem`, too.

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: topolvm-pvc
  namespace: default
  annotations:
    resize.topolvm.io/storage_limit: 100Gi
spec:
  accessModes:
  - ReadWriteOnce
  volumeMode: Filesystem
  resources:
    requests:
      storage: 30Gi
  storageClassName: topolvm-provisioner
```

The PVC can optionally have `resize.topolvm.io/threshold`, `resize.topolvm.io/inodes-threshold` and `resize.topolvm.io/increase` annotations.
(If they are not given, the default value is `10%`.)

When the amount of free space of the volume is below `resize.topolvm.io/threshold`
or the number of free inodes is below `resize.topolvm.io/inodes-threshold`,
`.spec.resources.requests.storage` is increased by `resize.topolvm.io/increase`.

If `resize.topolvm.io/increase` is given as a percentage, the value is calculated as
the current `spec.resources.requests.storage` value multiplied by the annotation value.

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: topolvm-pvc
  namespace: default
  annotations:
    resize.topolvm.io/storage_limit: 100Gi
    resize.topolvm.io/threshold: 20%
    resize.topolvm.io/inodes-threshold: 20%
    resize.topolvm.io/increase: 20Gi
spec:
  <snip>
```

#### Initial resize

PVC request size can also be changed at the creation time based on the largest PVC size in the same group. PVCs are grouped by labels, and the label key for grouping is specified by `resize.topolvm.io/initial-resize-group-by` annotation.

For example, suppose there are following three PVCs.

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

When the size of the largest PVC in the same group is larger than the value set to `resize.topolvm.io/storage_limit` annotation,
the PVC is resized up to this limit.

#### StatefulSet provisioned PersistentVolumeClaims

PVCs provisioned through a StatefulSet's `volumeClaimTemplates` cannot have their annotations updated automatically. `pvc-autoresizer` can be configured to automatically reconcile PVC annotations to match those found in the owning STS `volumeClaimTemplate` using the `--annotation-patching-enabled` flag. Additionally, this behavior is only enabled if the owning STS has the `resize.topolvm.io/annotation-patching-enabled: "true"` annotation. This reconciliation is only done for the following annotations: `resize.topolvm.io/threshold`, `resize.topolvm.io/inodes-threshold`, `resize.topolvm.io/increase`, `resize.topolvm.io/storage_limit`, and `resize.topolvm.io/initial-resize-group-by`.

For example, for the following existing STS and provisioned PVC:

```yaml
kind: StatefulSet
metadata:
  name: sts
  annotations:
    resize.topolvm.io/annotation-patching-enabled: "true"
spec:
  volumeClaimTemplates:
  - kind: PersistentVolumeClaim
    metadata:
      annotations:
        resize.topolvm.io/storage_limit: 20Gi
# ...
---
kind: PersistentVolumeClaim
metadata:
  annotations:
    resize.topolvm.io/storage_limit: 10Gi
#...
```

When `pvc-autoresizer` attempts to resize the PVC, it will first update the storage limit annotation value to `20Gi` to match the STS template.

### Prometheus metrics

####  `pvcautoresizer_kubernetes_client_fail_total`

`pvcautoresizer_kubernetes_client_fail_total` is a counter that indicates how many API requests to kube-api server are failed.

#### `pvcautoresizer_metrics_client_fail_total`

`pvcautoresizer_metrics_client_fail_total` is a counter that indicates how many API requests to metrics server(e.g. prometheus) are failed.

#### `pvcautoresizer_loop_seconds_total`

`pvcautoresizer_loop_seconds_total` is a counter that indicates the sum of seconds spent on volume expansion processing loops.

####  `pvcautoresizer_success_resize_total`

`pvcautoresizer_success_resize_total` is a counter that indicates how many volume expansion processing resizes succeed.

####  `pvcautoresizer_failed_resize_total`

`pvcautoresizer_failed_resize_total` is a counter that indicates how many volume expansion processing resizes fail.

####  `pvcautoresizer_limit_reached_total`

`pvcautoresizer_limit_reached_total` is a counter that indicates how many storage limit was reached.

## Contributing

pvc-autoresizer project welcomes contributions from any member of our community. To get
started contributing, please see our [Contributor Guide](CONTRIBUTING.md).

## Communications

If you have any questions or ideas, please use [discussions](https://github.com/topolvm/topolvm/discussions).

## Resources

[docs](docs/) directory contains designs, and so on.

## License

This project is licensed under [Apache License 2.0](LICENSE).

[releases]: https://github.com/topolvm/pvc-autoresizer/releases
