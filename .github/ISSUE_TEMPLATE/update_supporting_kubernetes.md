---
name: Update supporting Kubernetes
about: Dependencies relating to Kubernetes version upgrades
title: 'Update supporting Kubernetes'
labels: 'update kubernetes'
assignees: ''

---

## Update Procedure

- Read [this document](https://github.com/topolvm/pvc-autoresizer/blob/main/docs/maintenance.md).

## Before Check List

There is a check list to confirm depending libraries or tools are released.

### Must Update Dependencies

Must update Kubernetes with each new version of Kubernetes.

- [ ] k8s.io/api
  - https://github.com/kubernetes/api/tags
    - The supported Kubernetes version is written in the description of each tag.
- [ ] k8s.io/apimachinery
  - https://github.com/kubernetes/apimachinery/tags
    - The supported Kubernetes version is written in the description of each tag.
- [ ] k8s.io/client-go
  - https://github.com/kubernetes/client-go/tags
    - The supported Kubernetes version is written in the description of each tag.
- [ ] sigs.k8s.io/controller-runtime
  - https://github.com/kubernetes-sigs/controller-runtime/releases
- [ ] sigs.k8s.io/controller-tools
  - https://github.com/kubernetes-sigs/controller-tools/releases
- [ ] topolvm
  - https://github.com/topolvm/topolvm/blob/main/CHANGELOG.md

## Checklist

- [ ] Finish implementation of the issue
- [ ] Test all functions
