# parameters
TEST_KUBERNETES_TARGET ?= current

ifeq ($(TEST_KUBERNETES_TARGET),current)
KUBERNETES_VERSION := 1.21.2
else ifeq ($(TEST_KUBERNETES_TARGET),prev)
KUBERNETES_VERSION := 1.20.7
else ifeq ($(TEST_KUBERNETES_TARGET),prev2)
KUBERNETES_VERSION := 1.19.11
endif
export KUBERNETES_VERSION

KUBE_PROMETHEUS_VERSION := 0.8.0
TOPOLVM_VERSION := 3f3854c808df5d68f46ba316e96bfe6bb7edcd48
KIND_NODE_IMAGE=kindest/node:v$(KUBERNETES_VERSION)
KIND_CLUSTER_NAME := autoresizer-e2e

SUDO := sudo
BINDIR := $(shell pwd)/../bin
TMPDIR := /tmp/autoresizer
KUBECTL := $(BINDIR)/kubectl
KUSTOMIZE := $(BINDIR)/kustomize
GINKGO := $(BINDIR)/ginkgo
KIND := $(TMPDIR)/topolvm/bin/kind

.PHONY: setup
setup:
	$(MAKE) -C ../ setup
	mkdir -p $(BINDIR)
	curl -o $(BINDIR)/kubectl -sfL https://storage.googleapis.com/kubernetes-release/release/v$(KUBERNETES_VERSION)/bin/linux/amd64/kubectl
	chmod a+x $(BINDIR)/kubectl
	GOBIN=$(BINDIR) go install github.com/onsi/ginkgo/ginkgo@latest

.PHONY: init-cluster
init-cluster: launch-kind autoresizer.img kube-prometheus
	# https://github.com/prometheus-operator/kube-prometheus/tree/v$(KUBE_PROMETHEUS_VERSION)#quickstart"
	$(KUBECTL) apply -f kube-prometheus/manifests/setup
	until $(KUBECTL) get servicemonitors --all-namespaces ; do date; sleep 1; echo ""; done
	$(KUBECTL) apply -f kube-prometheus/manifests/
	# setup autoresizer
	$(KIND) load image-archive --name=$(KIND_CLUSTER_NAME) autoresizer.img
	$(KUSTOMIZE) build --load_restrictor='none' $(shell pwd)/manifests/pvc-autoresizer | $(KUBECTL) apply -f -
	# storageclass for test
	$(KUBECTL) apply -f manifests/storageclass.yaml

.PHONY: test
test:
	E2ETEST=1 BINDIR=$(BINDIR) $(GINKGO) --failFast -v .

.PHONY: launch-kind
launch-kind: $(TMPDIR)/topolvm
	cp autoresizer-cluster.yaml $(TMPDIR)/topolvm/example/kind/topolvm-cluster.yaml
	make -C $(TMPDIR)/topolvm/example run \
		KUBERNETES_VERSION=$(KUBERNETES_VERSION) \
		KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) \
		TMPDIR=$(TMPDIR)

.PHONY: shutdown-kind
shutdown-kind:
	$(call call-topolvm-make-task,$@)

.PHONY: stop-lvmd
stop-lvmd:
	$(call call-topolvm-make-task,$@)

define call-topolvm-make-task
if [ -d $(TMPDIR)/topolvm/example ]; then make -C $(TMPDIR)/topolvm/example $(1); fi
endef

$(TMPDIR)/topolvm:
	git clone https://github.com/topolvm/topolvm.git $@
	cd $@ && git checkout $(TOPOLVM_VERSION)
	make -C $(TMPDIR)/topolvm/example setup

autoresizer.img:
	IMAGE_PREFIX="" $(MAKE) -C ../ image
	docker save -o $@ pvc-autoresizer:devel

kube-prometheus:
	wget https://github.com/prometheus-operator/kube-prometheus/archive/refs/tags/v$(KUBE_PROMETHEUS_VERSION).tar.gz -O - \
		|  tar xvfz - -C $(shell pwd) \
		&& mkdir $(shell pwd)/kube-prometheus \
		&& mv $(shell pwd)/kube-prometheus-$(KUBE_PROMETHEUS_VERSION)/manifests kube-prometheus/manifests \
		&& rm -fr $(shell pwd)/kube-prometheus-$(KUBE_PROMETHEUS_VERSION)

.PHONY: clean
clean: stop-lvmd
	rm -rf \
		$(BINDIR) \
		$(TMPDIR) \
		autoresizer.img \
		kube-prometheus
