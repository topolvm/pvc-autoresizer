[![GitHub release](https://img.shields.io/github/v/release/topolvm/pvc-autoresizer.svg?maxAge=60)][releases]
[![Main](https://github.com/topolvm/pvc-autoresizer/workflows/Main/badge.svg)](https://github.com/topolvm/pvc-autoresizer/actions)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/topolvm/pvc-autoresizer?tab=overview)](https://pkg.go.dev/github.com/topolvm/pvc-autoresizer?tab=overview)
[![Go Report Card](https://goreportcard.com/badge/github.com/topolvm/pvc-autoresizer)](https://goreportcard.com/badge/github.com/topolvm/pvc-autoresizer)

# pvc-autoresizer

`pvc-autoresizer` resizes PersistentVolumeClaims (PVCs) when the free amount of storage is below the threshold.
It queries the volume usage metrics from Prometheus that collects metrics from `kubelet`.

**Status**: beta

## Supported environments

- Kubernetes: 1.24, 1.23, 1.22

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

`pvc-autoresizer` can be deployed to a Kubernetes cluster via `helm`:

```sh
helm install --create-namespace --namespace pvc-autoresizer pvc-autoresizer pvc-autoresizer/pvc-autoresizer --set "controller.args.prometheusURL=<YOUR PROMETHEUS ENDPOINT>"
```

See the Chart [README.md](./charts/pvc-autoresizer/README.md) for detailed documentation on the Helm Chart.

## How to use

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
provisioner: topolvm.cybozu.com
allowVolumeExpansion: true
```

To allow auto volume expansion, the PVC to be resized need to specify the upper limit of
volume size with the annotation `resize.topolvm.io/storage_limit`. The PVC must have `volumeMode: Filesystem` too.
You can also set the upper limit of volume size with `.spec.resources.limits.storage`, but it is deprecated. If both are present, the annotation takes precedence.

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

## Container images

Container images are available on [ghcr.io](https://github.com/topolvm/pvc-autoresizer/pkgs/container/pvc-autoresizer)  
**Please note that [the images on quay.io](https://quay.io/repository/topolvm/pvc-autoresizer) are currently deprecated and will no longer be available in the future.**

[releases]: https://github.com/topolvm/pvc-autoresizer/releases

## Prometheus metrics

###  `pvcautoresizer_kubernetes_client_fail_total`

`pvcautoresizer_kubernetes_client_fail_total` is a counter that indicates how many API requests to kube-api server are failed.

### `pvcautoresizer_metrics_client_fail_total`

`pvcautoresizer_metrics_client_fail_total` is a counter that indicates how many API requests to metrics server(e.g. prometheus) are failed.

### `pvcautoresizer_loop_seconds_total`

`pvcautoresizer_loop_seconds_total` is a counter that indicates the sum of seconds spent on volume expansion processing loops.

###  `pvcautoresizer_success_resize_total`

`pvcautoresizer_success_resize_total` is a counter that indicates how many volume expansion processing resizes succeed.

###  `pvcautoresizer_failed_resize_total`

`pvcautoresizer_failed_resize_total` is a counter that indicates how many volume expansion processing resizes fail.

###  `pvcautoresizer_limit_reached_total`

`pvcautoresizer_limit_reached_total` is a counter that indicates how many storage limit was reached.
