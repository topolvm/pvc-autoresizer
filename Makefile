# Makefile for pvc-autoresizer

K8S_VERSION = 1.18.9
KUBEBUILDER_VERSION = 2.3.1
CTRLTOOLS_VERSION = 0.5.0
KUSTOMIZE_VERSION = 3.7.0

## DON'T EDIT BELOW THIS LINE
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

CRD_OPTIONS = "crd:crdVersions=v1"

BINDIR := $(PWD)/bin
CONTROLLER_GEN := $(BINDIR)/controller-gen
KUBEBUILDER_ASSETS := $(BINDIR)
export KUBEBUILDER_ASSETS

IMAGE_TAG ?= latest
IMAGE_PREFIX ?= quay.io/topolvm/

.PHONY: all
all: manager

.PHONY: test
test: tools
	test -z "$$(gofmt -s -l . | tee /dev/stderr)"
	staticcheck ./...
	test -z "$$(nilerr ./... 2>&1 | tee /dev/stderr)"
	go install ./...
	go test -race -v -count 1 ./...
	go vet ./...

.PHONY: manager
manager: generate
	go build -o bin/manager main.go

.PHONY: run
run: generate manifests
	go run ./main.go

.PHONY: deploy
deploy: manifests
	cd config/manager && kustomize edit set image pvc-autoresizer=$(IMAGE_PREFIX)pvc-autoresizer:devel
	kustomize build config/default | kubectl apply -f -

.PHONY: manifests
manifests:
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=controller webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate:
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: image
image:
	docker build . -t $(IMAGE_PREFIX)pvc-autoresizer:devel

.PHONY: tag
tag:
	docker tag $(IMAGE_PREFIX)pvc-autoresizer:devel $(IMAGE_PREFIX)pvc-autoresizer:$(IMAGE_TAG)

.PHONY: push
push:
	docker push $(IMAGE_PREFIX)pvc-autoresizer:$(IMAGE_TAG)

.PHONY: tools
tools: staticcheck nilerr

.PHONY: staticcheck
staticcheck:
	if ! which staticcheck >/dev/null; then \
		env GOFLAGS= go install honnef.co/go/tools/cmd/staticcheck@latest; \
	fi

.PHONY: nilerr
nilerr:
	if ! which nilerr >/dev/null; then \
		env GOFLAGS= go install github.com/gostaticanalysis/nilerr/cmd/nilerr@latest; \
	fi

.PHONY: setup
setup:
	mkdir -p bin
	curl -sfL https://go.kubebuilder.io/dl/$(KUBEBUILDER_VERSION)/$(GOOS)/$(GOARCH) | tar -xz -C /tmp/
	mv /tmp/kubebuilder_$(KUBEBUILDER_VERSION)_$(GOOS)_$(GOARCH)/bin/* bin/
	rm -rf /tmp/kubebuilder_*
	GOBIN=$(BINDIR) go install sigs.k8s.io/controller-tools/cmd/controller-gen@v$(CTRLTOOLS_VERSION)
	curl -o $(BINDIR)/kubectl -sfL https://storage.googleapis.com/kubernetes-release/release/v$(K8S_VERSION)/bin/linux/amd64/kubectl
	chmod a+x $(BINDIR)/kubectl
	curl -sfL https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv$(KUSTOMIZE_VERSION)/kustomize_v$(KUSTOMIZE_VERSION)_linux_amd64.tar.gz | tar -xz -C $(BINDIR)
