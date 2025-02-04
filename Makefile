# Makefile for pvc-autoresizer
include versions.mk

## DON'T EDIT BELOW THIS LINE
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

CRD_OPTIONS = "crd:crdVersions=v1"

BINDIR := $(shell pwd)/bin
CONTROLLER_GEN := $(BINDIR)/controller-gen
GOLANGCI_LINT = $(BINDIR)/golangci-lint
KUBECTL := $(BINDIR)/kubectl
KUSTOMIZE := $(BINDIR)/kustomize
SETUP_ENVTEST := $(BINDIR)/setup-envtest

KUBEBUILDER_ASSETS := $(shell $(SETUP_ENVTEST) use -p path $(ENVTEST_K8S_VERSION))
export KUBEBUILDER_ASSETS

IMAGE_TAG ?= latest
IMAGE_PREFIX ?= ghcr.io/topolvm/

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=controller webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate-helm-docs
generate-helm-docs:
	./bin/helm-docs -c charts/pvc-autoresizer/

.PHONY: generate
generate: manifests generate-helm-docs

.PHONY: check-uncommitted
check-uncommitted: generate ## Check if latest generated artifacts are committed.
	git diff --exit-code --name-only

.PHONY: fmt
fmt: ## Run go fmt against code.
	test -z "$$(gofmt -s -l . | tee /dev/stderr)"

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate tools fmt vet ## Run tests.
	$(shell go env GOPATH)/bin/staticcheck ./...
	go install ./...
	source <($(SETUP_ENVTEST) use -p env $(ENVTEST_K8S_VERSION)); \
		go test -race -v -count 1 ./... --timeout=60s

.PHONY: lint
lint: ## Run golangci-lint linter & yamllint
	$(GOLANGCI_LINT) run --timeout 3m

.PHONY: lint-fix
lint-fix: ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

##@ Build

.PHONY: build
build: ## Build manager binary.
	go build -o $(BINDIR)/manager ./cmd/*

.PHONY: run
run: manifests generate ## Run a controller from your host.
	go run ./cmd/main.go

.PHONY: image
image: ## Build docker image.
	docker build . -t $(IMAGE_PREFIX)pvc-autoresizer:devel

.PHONY: tag
tag: ## Set a docker tag to the image.
	docker tag $(IMAGE_PREFIX)pvc-autoresizer:devel $(IMAGE_PREFIX)pvc-autoresizer:$(IMAGE_TAG)

.PHONY: push
push: ## Push docker image.
	docker push $(IMAGE_PREFIX)pvc-autoresizer:$(IMAGE_TAG)

##@ Chart Testing

.PHONY: ct-lint
ct-lint: ## Lint and validate a chart.
	docker run \
		--rm \
		--user $(shell id -u $(USER)) \
		--workdir=/data \
		--volume $(shell pwd):/data \
		quay.io/helmpack/chart-testing:v$(CHART_TESTING_VERSION) \
		ct lint --config ct.yaml

.PHONY: ct-install
ct-install: ## Install and test a chart.
	docker run \
		--rm \
		--user $(shell id -u $(USER)) \
		--network host \
		--workdir=/data \
		--env KUBECONFIG=/kubeconfig \
		--volume ~/.kube/config:/kubeconfig:ro \
		--volume $(shell pwd):/data \
		quay.io/helmpack/chart-testing:v$(CHART_TESTING_VERSION) \
		ct install --config ct.yaml

##@ Tools

.PHONY: tools
tools: staticcheck setup-envtest

.PHONY: staticcheck
staticcheck: ## Install staticcheck
	if ! which staticcheck >/dev/null; then \
		env GOFLAGS= go install honnef.co/go/tools/cmd/staticcheck@latest; \
	fi

.PHONY: setup-envtest
setup-envtest: $(SETUP_ENVTEST) ## Download setup-envtest locally if necessary
$(SETUP_ENVTEST):
	# see https://github.com/kubernetes-sigs/controller-runtime/tree/master/tools/setup-envtest
	GOBIN=$(BINDIR) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(ENVTEST_BRANCH)

.PHONY: setup
setup: # Setup tools
	mkdir -p bin
	GOBIN=$(BINDIR) go install sigs.k8s.io/controller-tools/cmd/controller-gen@v$(CONTROLLER_TOOLS_VERSION)
	curl -o $(KUBECTL) -sSfL https://dl.k8s.io/release/v$(KUBERNETES_VERSION)/bin/linux/amd64/kubectl
	chmod a+x $(KUBECTL)
	GOBIN=$(BINDIR) go install github.com/norwoodj/helm-docs/cmd/helm-docs@v$(HELM_DOCS_VERSION)
	curl -sSfL https://get.helm.sh/helm-v$(HELM_VERSION)-linux-amd64.tar.gz \
	  | tar xvz -C $(BINDIR) --strip-components 1 linux-amd64/helm
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell dirname $(GOLANGCI_LINT)) $(GOLANGCI_LINT_VERSION)
