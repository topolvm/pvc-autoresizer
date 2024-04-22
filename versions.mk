# https://github.com/helm/chart-testing/releases
CHART_TESTING_VERSION := 3.10.1
# https://github.com/kubernetes-sigs/controller-tools/releases
CONTROLLER_TOOLS_VERSION := 0.13.0
# https://github.com/golangci/golangci-lint/releases
GOLANGCI_LINT_VERSION := v1.55.2
# https://github.com/norwoodj/helm-docs/releases
HELM_DOCS_VERSION := 1.12.0
# https://github.com/helm/helm/releases
HELM_VERSION := 3.14.0
# https://github.com/prometheus-operator/kube-prometheus/releases
KUBE_PROMETHEUS_VERSION := 0.13.0
# It is set by CI using the environment variable, use conditional assignment.
KUBERNETES_VERSION ?= 1.28.0
TOPOLVM_VERSION := topolvm-chart-v14.0.0

# Tools versions which are defined in go.mod
SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
CONTROLLER_RUNTIME_VERSION := $(shell awk '/sigs\.k8s\.io\/controller-runtime/ {print substr($$2, 2)}' $(SELF_DIR)/go.mod)

ENVTEST_BRANCH := release-$(shell echo $(CONTROLLER_RUNTIME_VERSION) | cut -d "." -f 1-2)
ENVTEST_K8S_VERSION := $(shell echo $(KUBERNETES_VERSION) | cut -d "." -f 1-2)

# Tools versions which are defined in go.mod
SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GINKGO_VERSION := $(shell awk '/github.com\/onsi\/ginkgo\/v2/ {print substr($$2, 2)}' $(SELF_DIR)/go.mod)

ifeq ($(KUBERNETES_VERSION), 1.28.0)
KIND_NODE_IMAGE=kindest/node:v1.28.0@sha256:b7a4cad12c197af3ba43202d3efe03246b3f0793f162afb40a33c923952d5b31
else ifeq ($(KUBERNETES_VERSION),1.27.3)
KIND_NODE_IMAGE=kindest/node:v1.27.3@sha256:3966ac761ae0136263ffdb6cfd4db23ef8a83cba8a463690e98317add2c9ba72
else ifeq ($(KUBERNETES_VERSION),1.26.6)
KIND_NODE_IMAGE=kindest/node:v1.26.6@sha256:6e2d8b28a5b601defe327b98bd1c2d1930b49e5d8c512e1895099e4504007adb
endif
