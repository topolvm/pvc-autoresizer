Maintenance guide
=================

This is the maintenance guide for pvc-autoresizer.

How to upgrade supported Kubernetes version
-------------------------------------------

pvc-autoresizer depends on some Kubernetes repositories like `k8s.io/client-go` and should support 3 consecutive Kubernets versions at a time.
Here is the guide for how to upgrade the supported versions.
Issues and PRs related to the last upgrade task also help you understand how to upgrade the supported versions,
so checking them together with this guide is recommended when you do this task.

### Check release notes

First of all, we should have a look at the release notes in the order below.

1. Kubernetes
    - Choose the next version and check the [release note](https://kubernetes.io/docs/setup/release/notes/). e.g. 1.17, 1.18, 1.19 -> 1.18, 1.19, 1.20
    - Read the [release note](https://github.com/kubernetes-sigs/controller-runtime/releases), and check whether there are serious security fixes and whether the new minor version is compatible with older versions from the pvc-autoresizer's point of view. If there are breaking changes, we should decide how to manage these changes.
    - Read the [kubebuilder go.mod](https://github.com/kubernetes-sigs/kubebuilder/blob/master/go.mod), and check the controller-tools version corresponding to controller-runtime.
2. TopoLVM
    - Choose the [TopoLVM](https://github.com/topolvm/topolvm/releases) version that supported target Kubernetes version.
3. Depending tools
    - They does not depend on other software, use latest versions.
      - [helm](https://github.com/helm/helm/releases)
      - [helm-docs](github.com/norwoodj/helm-docs/releases)
      - [kube-prometheus](https://github.com/prometheus-operator/kube-prometheus/releases)
4. Depending modules
  - Read [kubernetes go.mod](https://github.com/kubernetes/kubernetes/blob/master/go.mod), and update the `prometheus/*` modules.
  - Read [csi-test go.mod](https://github.com/kubernetes-csi/csi-test/blob/master/go.mod), and update the `ginkgo` and `gomega` modules.

Please write down to the Github issue of this task what kinds of changes we find in the release note and what we are going to do and NOT going to do to address the changes.
The format is up to you, but this is very important to keep track of what changes are made in this task, so please do not forget to do it.

Basically, we should pay attention to breaking changes and security fixes first.
If we find some interesting features added in new versions, please consider if we are going to use them or not and make a GitHub issue to incorporate them after the upgrading task is done.

### Update written versions

Once we decide the versions we are going to upgrade, we should update the versions written in the following files manually.

- `README.md`: Documentation which indicates what versions are supported by pvc-autoresizer
- `Makefile`: Makefile for running envtest
- `e2e/Makefile`: Makefile for running e2e tests

`git grep 1.18`, `git grep image:`, and `git grep -i VERSION` might help us avoid overlooking necessary changes.
Please update the versions in the code and docs with great care.

### Update dependencies

Next, we should update `go.mod` by the following commands.
Please note that Kubernetes v1 corresponds with v0 for the release tags. For example, v1.17.2 corresponds with the `v0.17.2` tag.

```bash
$ VERSION=<upgrading Kubernetes release version>
$ go get k8s.io/api@v${VERSION} k8s.io/apimachinery@v${VERSION} k8s.io/client-go@v${VERSION}
```

If we need to upgrade the `controller-runtime` version, do the following as well.

```bash
$ VERSION=<upgrading controller-runtime version>
$ go get sigs.k8s.io/controller-runtime@v${VERSION}
```

Then, please tidy up the dependencies.

```bash
$ go mod tidy
```

These are minimal changes for the Kubernetes upgrade, but if there are some breaking changes found in the release notes, you have to handle them as well in this step.

### Release the changes

We should follow [RELEASE.md](../RELEASE.md) and upgrade the minor version.

### Prepare for the next upgrade

We should create an issue for the next upgrade. Besides, Please update this document if we find something to be updated.
