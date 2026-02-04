# https://github.com/helm/chart-testing/releases
CHART_TESTING_VERSION := 3.14.0
# https://github.com/kubernetes-sigs/controller-tools/releases
CONTROLLER_TOOLS_VERSION := 0.19.0
# https://github.com/golangci/golangci-lint/releases
GOLANGCI_LINT_VERSION := v2.8.0
# https://github.com/norwoodj/helm-docs/releases
HELM_DOCS_VERSION := 1.14.2
# https://github.com/helm/helm/releases
HELM_VERSION := 4.1.0
# https://github.com/prometheus-operator/kube-prometheus/releases
KUBE_PROMETHEUS_VERSION := 0.16.0
# It is set by CI using the environment variable, use conditional assignment.
KUBERNETES_VERSION ?= 1.34.3
TOPOLVM_VERSION := v0.39.0

# Tools versions which are defined in go.mod
SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
CONTROLLER_RUNTIME_VERSION := $(shell awk '/sigs\.k8s\.io\/controller-runtime/ {print substr($$2, 2)}' $(SELF_DIR)/go.mod)

ENVTEST_KUBERNETES_VERSION := $(shell echo $(KUBERNETES_VERSION) | cut -d "." -f 1-2).0

# Tools versions which are defined in go.mod
SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GINKGO_VERSION := $(shell awk '/github.com\/onsi\/ginkgo\/v2/ {print substr($$2, 2)}' $(SELF_DIR)/go.mod)
