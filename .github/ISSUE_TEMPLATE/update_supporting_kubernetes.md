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

There is a check list to confirm depending libraries or tools are released. The release notes for Kubernetes should also be checked.

### Must Update Dependencies

Must update Kubernetes with each new version of Kubernetes.

- [ ] sigs.k8s.io/controller-runtime
  - https://github.com/kubernetes-sigs/controller-runtime/releases
- [ ] sigs.k8s.io/controller-tools
  - https://github.com/kubernetes-sigs/controller-tools/releases
- [ ] topolvm
  - https://github.com/topolvm/topolvm/blob/main/CHANGELOG.md

### Release notes check

- [ ] Read the necessary release notes for Kubernetes.

## Checklist

- [ ] Finish implementation of the issue
- [ ] Test all functions
