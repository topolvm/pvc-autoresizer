# Example

This directory contains manifests to run `pvc-autoresizer` in a demonstration environment.

## Setup TopoLVM

Deploy [TopoLVM] on [kind] as follows:

- https://github.com/topolvm/topolvm/tree/master/example

## Setup Prometheus

Deploy [Prometheus Operator] via [helm] as follows:

```
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install prometheus prometheus-community/kube-prometheus-stack --create-namespace --namespace=prometheus
```

## Deploy pvc-autoresizer

Load the container image of `pvc-autoresizer` to [kind] as follows:

```
make image
kind load docker-image --name topolvm-example ghcr.io/topolvm/pvc-autoresizer:devel
```

Deploy `pvc-autoresizer`:

```
helm repo add pvc-autoresizer https://topolvm.github.io/pvc-autoresizer/
helm install --create-namespace --namespace pvc-autoresizer pvc-autoresizer pvc-autoresizer/pvc-autoresizer --set "controller.args.prometheusURL=http://prometheus-kube-prometheus-prometheus.prometheus.svc:9090"
```

## Enable auto-resizing

Annotating a storage class enables the automatic resizing of PVCs it is associated with:

```
kubectl annotate storageclass topolvm-provisioner resize.topolvm.io/enabled=true
```

## Deploy a Pod

Deploy a Pod and PVC to be resized:

```
kubectl apply -f podpvc.yaml
```

## Resizing PVC

Enter into the target pod:

```
kubectl exec -it example-pod bash
```

Make sure current volume usage:

```
root@example-pod:/# df -h /test1
Filesystem                                         Size  Used Avail Use% Mounted on
/dev/topolvm/8ad1c617-e572-4d0d-b4e8-d66e5a572df9 1014M   34M  981M   4% /test1
```

Create a file that take up 90% of the volume:

```
fallocate -l 900M /test1/test.txt
```

Make sure current volume usage again:

```
root@example-pod:/# df -h /test1
Filesystem                                         Size  Used Avail Use% Mounted on
/dev/topolvm/8ad1c617-e572-4d0d-b4e8-d66e5a572df9 1014M  934M   81M  93% /test1
```

After a few minutes, the volume will be resized to 2GiB:

```
root@example-pod:/# df -h /test1
Filesystem                                         Size  Used Avail Use% Mounted on
/dev/topolvm/8ad1c617-e572-4d0d-b4e8-d66e5a572df9  2.0G  935M  1.1G  46% /test1
```

[TopoLVM]: https://github.com/topolvm/topolvm/
[Prometheus Operator]: https://github.com/prometheus-operator/prometheus-operator
[Helm]: https://helm.sh/
[kind]: https://github.com/kubernetes-sigs/kind
