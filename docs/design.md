# Design Notes


## Motivation

PersistentVolume(PV) is used for persist the data (ex. MySQL, Elasticsearch).
These data size will increase in the future.
It is difficult to estimate these in advance.
So, PV should be automatically expanded based on the PV usage.

## Goal

- Automatic resizing PersistentVolumeClaim(PVC).
- Allow users to set parameter for automatically resizing using PVC annotations.
- The target storage class to be resized can be specified using its annotations.

## Target

- CSI driver which supports [`VolumeExpansion`](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#csi-volume-expansion) (ex. TopoLVM, Ceph-CSI).
- Only FileSystem volume mode (not Raw).

## Architecture

![component diagram](http://www.plantuml.com/plantuml/svg/TP9DQyCm38Rl_XKYEwUmZrCOewMdNaQWqClOGKtKrCpvm9BRTQF_VNRgBDvXbnXBxoizYhnaGIkkDQhhQu9N__bM06yVRa-6v1tkZ6wEiZUEVBX6mJqomLPwYmt5x8MCwS_ggjIl00VDP4za0HcoLRc1spLB2aBePAaIx1f3C9ogKLoIPSr2dUnwurfQ6zIjzyKkgOLlZaZZXSopyAfc8Jel8TPV4QZShM4rnMQgnX9rYQsqVKjo9CS9TfAlMDTMJrEkjnkuNMS8bPI0t0tv2yHV2r10um-VjRfY5SP_pkl-tEKfRW7tYr4dQCFXoLab-LWqk0juMW1z3jZLm7515GvOQRdyjHWwY3UbR0LaZuiK2Fh3M4NICbd4z3sJuGiuerJ7mARcg6ymFPDYmZgD6rNypwWF8y4C7nOQn7cOJosfgrrhVW00)

### How pvc-autoresizer works

To expand PVC, pvc-autoresizer works as follows:

1. Get target PVC information from API server.
2. Get StorageClass related to the PVC from API server.
3. Get metrics from Prometheus about storage.
4. Expand PVC storage request size if PVC has less than the specified amount of free filesystem capacity. 

### Details

To enable pvc-autoresizer, prepare StorageClass as follows:

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

- To allow automatic resizing, StorageClass must have `resize.topolvm.io/enabled` annotation. 
- `allowVolumeExpansion` should be `true`.
- `csi.storage.k8s.io/fstype` in `parameters` should be set.

In addition to the StorageClass, prepare PVC as follows:

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

- `spec.storageClassName` should be put above StorageClass (in this case "topolvm-provisioner").
- To allow automatic resizing, PVC must have `spec.resources.limits.storage`.
- pvc-autoresizer increases PVC's `spec.resources.requests.storage` up to the given limits.
- The threshold of free space is given by `resize.topolvm.io/threshold` annotation.
- The amount of increased size can be specified by `resize.topolvm.io/amount` annotation.
- The value of the annotations can be a ratio like 20% or a value like 10Gi.
- The default value for both threshold and amount is 10%.

