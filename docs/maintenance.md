Maintenance guide
=================

How to change the supported Kubernetes minor versions
-------------------------------------------

pvc-autoresizer depends on some Kubernetes repositories like `k8s.io/client-go` and should support 3 consecutive Kubernetes versions at a time.

Issues and PRs related to the last upgrade task also help you understand how to upgrade the supported versions,
so checking them together(e.g https://github.com/topolvm/pvc-autoresizer/pull/85) with this guide is recommended when you do this task.

### Upgrade procedure

We should write down in the github issue of this task what are the important changes and the required actions to manage incompatibilities if exist.
The format is up to you.

Basically, we should pay attention to breaking changes and security fixes first.

#### TopoLVM

Choose the [TopoLVM](https://github.com/topolvm/topolvm/releases) version that supports target Kubernetes version.

To change the version, edit `versions.mk`. If TopoLVM which supports the target Kubernetes version has not released yet, you can specify the commit hash instead of the tag.

#### Kubernetes

Choose the next version and check the [release note](https://kubernetes.io/docs/setup/release/notes/). e.g. 1.17, 1.18, 1.19 -> 1.18, 1.19, 1.20

To change the version, edit the following files.

- `.github/workflows/e2e.yaml`
- `README.md`
- `versions.mk`

We should also update go.mod. According to [the Kubebuilder documentation](https://book.kubebuilder.io/versions_compatibility_supportability), we should use versions compatible with Kubebuilder, so refer to the samples in the latest Kubebuilder testdata directory (e.g., https://github.com/kubernetes-sigs/kubebuilder/blob/v4.1.1/testdata/project-v4/go.mod#L8-L11 and https://github.com/kubernetes-sigs/kubebuilder/blob/v4.1.1/testdata/project-v4/Makefile#L162) to see which versions should be used.

First, update `k8s.io/*` libraries. Please note that Kubernetes v1 corresponds with v0 for the release tags. For example, v1.17.2 corresponds with the v0.17.2 tag.

```bash
$ VERSION=<upgrading Kubernetes release version>
$ go get k8s.io/api@v${VERSION} k8s.io/apimachinery@v${VERSION} k8s.io/client-go@v${VERSION}
```

Next, update controller-runtime by the following command. Before updating it, please read the [`controller-runtime`'s release note](https://github.com/kubernetes-sigs/controller-runtime/releases). If there are breaking changes, we should decide how to manage these changes.

```
$ VERSION=<upgrading controller-runtime version>
$ go get sigs.k8s.io/controller-runtime@v${VERSION}
```

Finally, update controller-tools. Before updating it, please read the [`controller-tools`'s release note](https://github.com/kubernetes-sigs/controller-tools/releases). If there are breaking changes, we should decide how to manage these changes.
To change the version, edit `versions.mk`.

#### Go

Choose the version compatible with Kubebuilder (e.g., https://github.com/kubernetes-sigs/kubebuilder/blob/v4.1.1/testdata/project-v4/go.mod#L3).

Edit the following files.

- `go.mod`
- `Dockerfile`

#### Depending tools

The following tools don't depend on other software, so use the latest versions.
To change their versions, edit `versions.mk`.
- [chart-testing](https://github.com/helm/chart-testing/releases)
- [golangci-lint](https://github.com/golangci/golangci-lint/releases)
- [helm-docs](https://github.com/norwoodj/helm-docs/releases)
- [helm](https://github.com/helm/helm/releases)
- [kube-prometheus](https://github.com/prometheus-operator/kube-prometheus/releases)

#### Depending modules

Please tidy up the dependencies.

```bash
$ go mod tidy
```

Regenerate manifests using new controller-tools.

```bash
$ make setup
$ make generate
```

#### Update Ubuntu

If the support term for using Ubuntu is about to expire, update the versions.

Unlike TopoLVM, there is no reason to use an old version of Ubuntu for pvc-autoresizer. We use the same update policy at the organizational level to ensure consistency.

#### Final check

`git grep <the kubernetes version which support will be dropped>`, `git grep image:`, `git grep -i VERSION` and looking `versions.mk` might help to avoid overlooking necessary changes.
