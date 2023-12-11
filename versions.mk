CHART_TESTING_VERSION := 3.9.0
CONTROLLER_TOOLS_VERSION := 0.12.1
HELM_DOCS_VERSION := 1.11.3
HELM_VERSION := 3.13.0
KUBE_PROMETHEUS_VERSION := 0.13.0
# It is set by CI using the environment variable, use conditional assignment.
KUBERNETES_VERSION ?= 1.27.3
TOPOLVM_VERSION := topolvm-chart-v13.0.0

ENVTEST_K8S_VERSION := $(shell echo $(KUBERNETES_VERSION) | cut -d "." -f 1-2)

# Tools versions which are defined in go.mod
SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GINKGO_VERSION := $(shell awk '/github.com\/onsi\/ginkgo\/v2/ {print substr($$2, 2)}' $(SELF_DIR)/go.mod)

ifeq ($(KUBERNETES_VERSION),1.27.3)
KIND_NODE_IMAGE=kindest/node:v1.27.3@sha256:3966ac761ae0136263ffdb6cfd4db23ef8a83cba8a463690e98317add2c9ba72
else ifeq ($(KUBERNETES_VERSION),1.26.6)
KIND_NODE_IMAGE=kindest/node:v1.26.6@sha256:6e2d8b28a5b601defe327b98bd1c2d1930b49e5d8c512e1895099e4504007adb
else ifeq ($(KUBERNETES_VERSION),1.25.11)
KIND_NODE_IMAGE=kindest/node:v1.25.11@sha256:227fa11ce74ea76a0474eeefb84cb75d8dad1b08638371ecf0e86259b35be0c8
endif
