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

We should also update go.mod by the following commands. Please note that Kubernetes v1 corresponds with v0 for the release tags. For example, v1.17.2 corresponds with the v0.17.2 tag.

```bash
$ VERSION=<upgrading Kubernetes release version>
$ go get k8s.io/api@v${VERSION} k8s.io/apimachinery@v${VERSION} k8s.io/client-go@v${VERSION}
```

Read the [`controller-runtime`'s release note](https://github.com/kubernetes-sigs/controller-runtime/releases), and update to the newest version that is compatible with all supported kubernetes versions. If there are breaking changes, we should decide how to manage these changes.

```
$ VERSION=<upgrading controller-runtime version>
$ go get sigs.k8s.io/controller-runtime@v${VERSION}
```

Read the [`controller-tools`'s release note](https://github.com/kubernetes-sigs/controller-tools/releases), and update to the newest version that is compatible with all supported kubernetes versions. If there are breaking changes, we should decide how to manage these changes. To change the version, edit `versions.mk`. 

#### Go

Choose the same version of Go [used by the latest Kubernetes](https://github.com/kubernetes/kubernetes/blob/master/go.mod) supported by pvc-autoresizer.

Edit the following files.

- `go.mod`
- `Dockerfile`

#### Depending tools

The following tools do not depend on other software, use latest versions.
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
