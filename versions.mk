# https://github.com/zizmorcore/zizmor/releases
ZIZMOR_VERSION := v1.26.1
# SHA256 checksum of the zizmor release tarball for verification
ZIZMOR_SHA256 := 8556289a64e7aaf2400cd516f61a471aa91c5902cc56ad96a82fd12f90c2ef73
# https://github.com/helm/chart-testing/releases
CHART_TESTING_VERSION := 3.14.0
# https://github.com/kubernetes-sigs/controller-tools/releases
CONTROLLER_TOOLS_VERSION := 0.20.1
# https://github.com/golangci/golangci-lint/releases
GOLANGCI_LINT_VERSION := v2.11.4
# https://github.com/rhysd/actionlint/releases
ACTIONLINT_VERSION := v1.7.12
# https://github.com/suzuki-shunsuke/ghalint/releases
GHALINT_VERSION := v1.5.6
# https://github.com/norwoodj/helm-docs/releases
HELM_DOCS_VERSION := 1.14.2
# https://github.com/helm/helm/releases
HELM_VERSION := 4.1.4
# https://github.com/prometheus-operator/kube-prometheus/releases
KUBE_PROMETHEUS_VERSION := 0.17.0
# It is set by CI using the environment variable, use conditional assignment.
KUBERNETES_VERSION ?= 1.35.4
TOPOLVM_VERSION := v0.41.0 

# Tools versions which are defined in go.mod
SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
CONTROLLER_RUNTIME_VERSION := $(shell awk '/sigs\.k8s\.io\/controller-runtime/ {print substr($$2, 2)}' $(SELF_DIR)/go.mod)

ENVTEST_KUBERNETES_VERSION := $(shell echo $(KUBERNETES_VERSION) | cut -d "." -f 1-2).0

# Tools versions which are defined in go.mod
SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GINKGO_VERSION := $(shell awk '/github.com\/onsi\/ginkgo\/v2/ {print substr($$2, 2)}' $(SELF_DIR)/go.mod)
