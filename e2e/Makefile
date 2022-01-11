# parameters
TEST_KUBERNETES_TARGET ?= current

ifeq ($(TEST_KUBERNETES_TARGET),current)
KUBERNETES_VERSION := 1.22.4
else ifeq ($(TEST_KUBERNETES_TARGET),prev)
KUBERNETES_VERSION := 1.21.2
else ifeq ($(TEST_KUBERNETES_TARGET),prev2)
KUBERNETES_VERSION := 1.20.7
endif
export KUBERNETES_VERSION

KUBE_PROMETHEUS_VERSION := 0.9.0
TOPOLVM_VERSION := 0.10.3
KIND_NODE_IMAGE=kindest/node:v$(KUBERNETES_VERSION)
KIND_CLUSTER_NAME := autoresizer-e2e

GINKGO_VERSION := $(shell awk '/github.com\/onsi\/ginkgo/ {print substr($$2, 2)}' ../go.mod)

SUDO := sudo
BINDIR := $(shell pwd)/../bin
TMPDIR := /tmp/autoresizer
KUBECTL := $(BINDIR)/kubectl
HELM := $(BINDIR)/helm
GINKGO := $(BINDIR)/ginkgo
KIND := $(TMPDIR)/topolvm/bin/kind

.PHONY: setup
setup:
	$(MAKE) -C ../ setup
	mkdir -p $(BINDIR)
	curl -o $(BINDIR)/kubectl -sfL https://storage.googleapis.com/kubernetes-release/release/v$(KUBERNETES_VERSION)/bin/linux/amd64/kubectl
	chmod a+x $(BINDIR)/kubectl
	GOBIN=$(BINDIR) go install github.com/onsi/ginkgo/ginkgo@v$(GINKGO_VERSION)

.PHONY: init-cluster
init-cluster: launch-kind autoresizer.img kube-prometheus
	# https://github.com/prometheus-operator/kube-prometheus/tree/v$(KUBE_PROMETHEUS_VERSION)#quickstart"
	$(KUBECTL) apply -f kube-prometheus/manifests/setup
	until $(KUBECTL) get servicemonitors --all-namespaces ; do date; sleep 1; echo ""; done
	$(KUBECTL) apply -f kube-prometheus/manifests/
	# setup autoresizer
	$(KIND) load image-archive --name=$(KIND_CLUSTER_NAME) autoresizer.img
	$(HELM) install --create-namespace --namespace=pvc-autoresizer pvc-autoresizer ../charts/pvc-autoresizer/ -f manifests/values/values.yaml
	# storageclass for test
	$(KUBECTL) apply -f manifests/common/storageclass.yaml

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
	cd $@ && git checkout v$(TOPOLVM_VERSION)
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
