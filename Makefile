# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
GOFLAGS = -mod=vendor
export GOFLAGS

PACKAGES := unzip
KUBEBUILDER_VERSION = 2.3.1
CTRLTOOLS_VERSION = 0.3.0

SUDO=sudo

IMAGE_TAG ?= latest
IMAGE_PREFIX ?= quay.io/topolvm/

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
test: generate manifests
	test -z "$$(gofmt -s -l . | grep -v '^vendor' | tee /dev/stderr)"
	test -z "$$(golint $$(go list ./... | grep -v /vendor/) | tee /dev/stderr)"
	test -z "$$(nilerr ./... 2>&1 | tee /dev/stderr)"
	test -z "$$(custom-checker -restrictpkg.packages=html/template,log $$(go list -tags='$(GOTAGS)' ./... | grep -v /vendor/ ) 2>&1 | tee /dev/stderr)"
	ineffassign .
	go install ./...
	go test -race -v ./...
	go vet ./...
	test -z "$$(go vet ./... | grep -v '^vendor' | tee /dev/stderr)"

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image pvc-autoresizer=$(IMAGE_PREFIX)pvc-autoresizer:devel
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests:
	controller-gen $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate:
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
image:
	docker build . -t $(IMAGE_PREFIX)pvc-autoresizer:devel

tag:
	docker tag $(IMAGE_PREFIX)pvc-autoresizer:devel $(IMAGE_PREFIX)pvc-autoresizer:$(IMAGE_TAG)

# Push the docker image
push:
	docker push $(IMAGE_PREFIX)pvc-auto-resizer:$(IMAGE_TAG)

tools:
	cd /tmp; env GOFLAGS= GO111MODULE=on go get golang.org/x/tools/cmd/goimports
	cd /tmp; env GOFLAGS= GO111MODULE=on go get golang.org/x/lint/golint
	cd /tmp; env GOFLAGS= GO111MODULE=on go get github.com/gordonklaus/ineffassign
	cd /tmp; env GOFLAGS= GO111MODULE=on go get github.com/gostaticanalysis/nilerr/cmd/nilerr
	cd /tmp; env GOFLAGS= GO111MODULE=on go get github.com/cybozu/neco-containers/golang/analyzer/cmd/...

setup: tools
	$(SUDO) apt-get update
	$(SUDO) apt-get -y install --no-install-recommends $(PACKAGES)
	curl -sL https://go.kubebuilder.io/dl/$(KUBEBUILDER_VERSION)/$(GOOS)/$(GOARCH) | tar -xz -C /tmp/
	$(SUDO) rm -rf /usr/local/kubebuilder
	$(SUDO) mv /tmp/kubebuilder_$(KUBEBUILDER_VERSION)_$(GOOS)_$(GOARCH) /usr/local/kubebuilder
	$(SUDO) curl -o /usr/local/kubebuilder/bin/kustomize -sL https://go.kubebuilder.io/kustomize/$(GOOS)/$(GOARCH)
	$(SUDO) chmod a+x /usr/local/kubebuilder/bin/kustomize
	cd /tmp; env GOFLAGS= GO111MODULE=on go get sigs.k8s.io/controller-tools/cmd/controller-gen@v$(CTRLTOOLS_VERSION)

.PHONY: all test run deploy manifests generate image tag push tools setup
