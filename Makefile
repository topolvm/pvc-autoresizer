# Makefile for pvc-autoresizer

K8S_VERSION = 1.18.9
KUBEBUILDER_VERSION = 2.3.1
KUSTOMIZE_VERSION = 3.7.0

## DON'T EDIT BELOW THIS LINE
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
GO111MODULE=on
export GO111MODULE

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
		cd /tmp; env GOFLAGS= GO111MODULE=on go get honnef.co/go/tools/cmd/staticcheck; \
	fi

.PHONY: nilerr
nilerr:
	if ! which nilerr >/dev/null; then \
		cd /tmp; env GOFLAGS= GO111MODULE=on go get github.com/gostaticanalysis/nilerr/cmd/nilerr; \
	fi

.PHONY: setup
setup:
	mkdir -p bin
	curl -sfL https://go.kubebuilder.io/dl/$(KUBEBUILDER_VERSION)/$(GOOS)/$(GOARCH) | tar -xz -C /tmp/
	mv /tmp/kubebuilder_$(KUBEBUILDER_VERSION)_$(GOOS)_$(GOARCH)/bin/* bin/
	rm -rf /tmp/kubebuilder_*
	GOBIN=$(BINDIR) go install sigs.k8s.io/controller-tools/cmd/controller-gen
	# Replace bundled kube-apiserver with that of the minimal supported version
	rm -rf tmp && mkdir -p tmp
	curl -sfL https://github.com/kubernetes/kubernetes/archive/v$(K8S_VERSION).tar.gz | tar zxf - -C tmp
	mv tmp/kubernetes-$(K8S_VERSION) tmp/kubernetes
	cd tmp/kubernetes; make all WHAT="cmd/kube-apiserver"
	mv tmp/kubernetes/_output/bin/kube-apiserver bin/
	rm -rf tmp
	curl -o $(BINDIR)/kubectl -sfL https://storage.googleapis.com/kubernetes-release/release/v$(K8S_VERSION)/bin/linux/amd64/kubectl
	chmod a+x $(BINDIR)/kubectl
	curl -sfL https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv$(KUSTOMIZE_VERSION)/kustomize_v$(KUSTOMIZE_VERSION)_linux_amd64.tar.gz | tar -xz -C $(BINDIR)
