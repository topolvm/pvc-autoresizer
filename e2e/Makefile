include ../versions.mk

KIND_CLUSTER_NAME := autoresizer-e2e

export KUBERNETES_VERSION

GINKGO_VERSION := $(shell awk '/github.com\/onsi\/ginkgo\/v2/ {print substr($$2, 2)}' ../go.mod)

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
	GOBIN=$(BINDIR) go install github.com/onsi/ginkgo/v2/ginkgo@v$(GINKGO_VERSION)

.PHONY: init-cluster
init-cluster: launch-kind autoresizer.img kube-prometheus
	# https://github.com/prometheus-operator/kube-prometheus/tree/v$(KUBE_PROMETHEUS_VERSION)#quickstart"
	# add options to avoid applying error (https://github.com/prometheus-community/helm-charts/issues/1500#issuecomment-969149744)
	$(KUBECTL) apply -f kube-prometheus/manifests/setup --force-conflicts=true --server-side
	until $(KUBECTL) get servicemonitors --all-namespaces ; do date; sleep 1; echo ""; done
	$(KUBECTL) apply -f kube-prometheus/manifests/
	# prometheus-adapter is required to get "metrics.k8s.io/v1beta1"
	$(KUBECTL) wait deployment -n monitoring prometheus-adapter --for=condition=Available=True --timeout=120s
	sleep 10
	# setup autoresizer
	$(KIND) load image-archive --name=$(KIND_CLUSTER_NAME) autoresizer.img
	$(HELM) repo add jetstack https://charts.jetstack.io
	$(HELM) repo update
	$(HELM) dependency build ../charts/pvc-autoresizer/
	$(HELM) install --create-namespace --namespace=pvc-autoresizer pvc-autoresizer ../charts/pvc-autoresizer/ -f manifests/values/values.yaml
	# storageclass for test
	$(KUBECTL) apply -f manifests/common/storageclass.yaml

.PHONY: test
test:
	E2ETEST=1 BINDIR=$(BINDIR) $(GINKGO) --fail-fast -v .

.PHONY: launch-kind
launch-kind: $(TMPDIR)/topolvm
	cp autoresizer-cluster.yaml $(TMPDIR)/topolvm/example/kind/topolvm-cluster.yaml
	make -C $(TMPDIR)/topolvm/example run \
		KUBERNETES_VERSION=$(KUBERNETES_VERSION) \
		KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) \
		TMPDIR=$(TMPDIR)
	$(KUBECTL) delete pod my-pod-thin
	$(KUBECTL) delete pvc topolvm-pvc-thin
	$(KUBECTL) delete sc topolvm-provisioner-thin
	sudo lvremove -f myvg1/thinpool

.PHONY: shutdown-kind
shutdown-kind:
	$(call call-topolvm-make-task,$@)

.PHONY: stop-lvmd
stop-lvmd:
	$(call call-topolvm-make-task,$@)

define call-topolvm-make-task
if [ -d $(TMPDIR)/topolvm/example ]; then make -C $(TMPDIR)/topolvm/example $(1) KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) TMPDIR=$(TMPDIR); fi
endef

$(TMPDIR)/topolvm:
	git clone https://github.com/topolvm/topolvm.git $@
	cd $@ && git checkout $(TOPOLVM_VERSION)
	make -C $(TMPDIR)/topolvm/example setup

autoresizer.img:
	IMAGE_PREFIX="" $(MAKE) -C ../ image
	docker save -o $@ pvc-autoresizer:devel

kube-prometheus:
	mkdir $(shell pwd)/kube-prometheus && \
	curl -sSfL https://github.com/prometheus-operator/kube-prometheus/archive/refs/tags/v$(KUBE_PROMETHEUS_VERSION).tar.gz \
		| tar xfz - -C $(shell pwd)/kube-prometheus --strip-components=1 kube-prometheus-$(KUBE_PROMETHEUS_VERSION)/manifests

.PHONY: clean
clean: stop-lvmd
	for f in $$($(SUDO) find $(TMPDIR) -type f); do \
		if $(SUDO) mountpoint -q $$f; then \
			$(SUDO) umount $$f; \
		fi; \
	done
	$(SUDO) rm -rf $(TMPDIR)/controller $(TMPDIR)/worker
	rm -rf \
		$(TMPDIR) \
		autoresizer.img \
		kube-prometheus
