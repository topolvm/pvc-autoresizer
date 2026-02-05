# Operator-Aware Resizing

## Overview

Operator-aware resizing enables pvc-autoresizer to work harmoniously with Kubernetes operators that manage PersistentVolumeClaims. Instead of directly modifying a PVC's storage request (which operators may treat as configuration drift), pvc-autoresizer can patch a Custom Resource field, allowing the operator to reconcile the PVC size.

## Problem Statement

When pvc-autoresizer directly patches a PVC managed by an operator:

1. **Configuration Drift**: The operator sees the PVC size differs from its spec and may treat it as an unintended change
2. **Reconciliation Failures**: Some operators (e.g., RabbitMQ) may fail to reconcile or enter error states
3. **Reversion**: The operator might revert the resize or attempt to "fix" what it perceives as drift
4. **Complex Workarounds**: Operators may require pod restarts or other manual intervention

## Solution

pvc-autoresizer can patch a field in the operator's Custom Resource instead of the PVC directly. The operator then reconciles the PVC size based on its own logic, maintaining the desired workflow.

## Security Model

Operator-aware resizing implements **defense-in-depth** security with three layers:

### Layer 1: RBAC (Helm-Generated)

The pvc-autoresizer controller must be granted explicit RBAC permissions to patch each Custom Resource type. RBAC rules are automatically generated from your Helm values.

### Layer 2: Resource Classes (Admin-Controlled Path Restriction)

Administrators define "resource classes" that specify exactly which CR types and which fields within those CRs can be patched. Users cannot specify arbitrary paths - they must reference an admin-defined class.

### Layer 3: Path Validation (Code Enforced)

As an additional safeguard, paths are validated at configuration load time to ensure they only target fields under `/spec/`.

### Configuration in values.yaml

```yaml
operatorAwareResizing:
  enabled: true
  resourceClasses:
    - name: "rabbitmq"
      apiGroup: "rabbitmq.com"
      apiVersion: "v1beta1"
      kind: "RabbitmqCluster"
      resource: "rabbitmqclusters"
      path: "/spec/persistence/storage"

    - name: "cnpg-data"
      apiGroup: "postgresql.cnpg.io"
      apiVersion: "v1"
      kind: "Cluster"
      resource: "clusters"
      path: "/spec/storage/size"

    - name: "cnpg-wal"
      apiGroup: "postgresql.cnpg.io"
      apiVersion: "v1"
      kind: "Cluster"
      resource: "clusters"
      path: "/spec/walStorage/size"
```

**Helm automatically generates:**
- RBAC rules for each resource type
- A ConfigMap containing the resource class definitions
- Volume mounts and arguments for the controller

### Why Resource Classes?

This approach provides **admin-controlled security**:

1. **Single source of truth**: values.yaml controls both RBAC and allowed paths
2. **No tenant path specification**: Users reference classes by name, not raw paths
3. **Prevents field tampering**: Only pre-approved paths can be patched
4. **Audit-friendly**: All allowed configurations are visible in one place

### Prerequisites

Before using operator-aware resizing, you must:

1. Enable the feature in values.yaml: `operatorAwareResizing.enabled: true`
2. Define resource classes for each CR type and path you want to allow
3. Run `helm upgrade` to apply the configuration

**If the feature is disabled or no resource classes are defined, operator-aware resizing is completely disabled.**

## How It Works

```
┌──────────────────────────────────────────────────────────────┐
│ 1. Metrics indicate low disk space on PVC                    │
└──────────────────────────────┬───────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────┐
│ 2. pvc-autoresizer checks for target-resource-class          │
└──────────────────────────────┬───────────────────────────────┘
                               │
                ┌──────────────┴──────────────┐
                │                             │
                ▼                             ▼
┌───────────────────────────┐   ┌───────────────────────────┐
│ Class annotation present? │   │ No class annotation       │
│ Look up class definition  │   │ Patch PVC directly        │
│ Patch CR at allowed path  │   │ (standard mode)           │
└─────────────┬─────────────┘   └───────────────────────────┘
              │
              ▼
┌──────────────────────────────────────────────────────────────┐
│ 3. Operator detects CR change and reconciles                 │
└──────────────────────────────────────────────────────────────┘
              │
              ▼
┌──────────────────────────────────────────────────────────────┐
│ 4. Operator updates PVC spec.resources.requests.storage      │
└──────────────────────────────────────────────────────────────┘
              │
              ▼
┌──────────────────────────────────────────────────────────────┐
│ 5. Kubernetes resizes the underlying volume                  │
└──────────────────────────────────────────────────────────────┘
```

## Configuration

### PVC Annotations

Add these annotations to your PVC to enable operator-aware resizing:

| Annotation | Required | Description | Example |
|-----------|----------|-------------|---------|
| `resize.topolvm.io/target-resource-class` | Yes | Name of the admin-defined resource class | `rabbitmq` |
| `resize.topolvm.io/target-resource-name` | Yes | Name of the target CR instance (must be in same namespace as PVC) | `my-rabbitmq` |
| `resize.topolvm.io/target-filter-value` | Conditional | Value for array filter placeholder in path (required when path contains `[key=?]`) | `tbs1` |

**Note**: All existing autoresizer annotations (`storage_limit`, `threshold`, `increase`, etc.) continue to work as normal.

### Resource Class Fields

Each resource class in values.yaml must specify:

| Field | Required | Description | Example |
|-------|----------|-------------|---------|
| `name` | Yes | Unique identifier (DNS label compatible) | `rabbitmq` |
| `apiGroup` | Yes | API group of the target CR | `rabbitmq.com` |
| `apiVersion` | Yes | API version of the target CR | `v1beta1` |
| `kind` | Yes | Kind of the target CR | `RabbitmqCluster` |
| `resource` | Yes | Plural resource name (for RBAC) | `rabbitmqclusters` |
| `path` | Yes | JSON pointer to storage field (must start with /spec/) | `/spec/persistence/storage` |

#### Path Syntax

The `path` field supports optional key-value filters for selecting array elements:

```
/spec/tablespaces[name=?]/storage/size
```

- **Simple path**: `/spec/storage/size` - direct field access
- **Filter with placeholder**: `/spec/tablespaces[name=?]/storage/size` - `?` is replaced by `target-filter-value` annotation
- **Filter with hardcoded value**: `/spec/tablespaces[name=tbs1]/storage/size` - for testing or single-element cases

Only one filter per path is allowed. The filter key and value are exact string matches.

## How to Add Annotations

### Using kubectl

The easiest way to add operator-aware resizing annotations is using `kubectl annotate`:

#### RabbitMQ Operator

```bash
kubectl annotate pvc persistence-my-rabbitmq-server-0 -n messaging \
  resize.topolvm.io/enabled="true" \
  resize.topolvm.io/storage_limit="100Gi" \
  resize.topolvm.io/threshold="20%" \
  resize.topolvm.io/increase="10Gi" \
  resize.topolvm.io/target-resource-class="rabbitmq" \
  resize.topolvm.io/target-resource-name="my-rabbitmq"
```

#### CloudNativePG

```bash
# Data PVC
kubectl annotate pvc my-postgres-1 -n database \
  resize.topolvm.io/enabled="true" \
  resize.topolvm.io/storage_limit="500Gi" \
  resize.topolvm.io/threshold="20%" \
  resize.topolvm.io/increase="10Gi" \
  resize.topolvm.io/target-resource-class="cnpg-data" \
  resize.topolvm.io/target-resource-name="my-postgres"

# WAL PVC (if using separate WAL volumes)
kubectl annotate pvc my-postgres-1-wal -n database \
  resize.topolvm.io/enabled="true" \
  resize.topolvm.io/storage_limit="200Gi" \
  resize.topolvm.io/threshold="20%" \
  resize.topolvm.io/increase="10Gi" \
  resize.topolvm.io/target-resource-class="cnpg-wal" \
  resize.topolvm.io/target-resource-name="my-postgres"

# Tablespace PVC (using filter syntax)
kubectl annotate pvc my-postgres-tbs1 -n database \
  resize.topolvm.io/enabled="true" \
  resize.topolvm.io/storage_limit="100Gi" \
  resize.topolvm.io/threshold="20%" \
  resize.topolvm.io/increase="10Gi" \
  resize.topolvm.io/target-resource-class="cnpg-tablespace" \
  resize.topolvm.io/target-resource-name="my-postgres" \
  resize.topolvm.io/target-filter-value="tbs1"
```

**Tips:**
- Use `--overwrite` flag if annotations already exist
- Verify annotations: `kubectl get pvc <name> -n <namespace> -o yaml | grep resize.topolvm.io`

### YAML Examples

#### RabbitMQ Operator

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: data-rabbitmq-0
  namespace: rabbitmq
  annotations:
    # Standard autoresizer annotations
    resize.topolvm.io/storage_limit: "100Gi"
    resize.topolvm.io/threshold: "20%"
    resize.topolvm.io/increase: "10Gi"

    # Operator-aware resizing annotations
    resize.topolvm.io/target-resource-class: "rabbitmq"
    resize.topolvm.io/target-resource-name: "my-rabbitmq"
spec:
  accessModes: ["ReadWriteOnce"]
  resources:
    requests:
      storage: 10Gi
  storageClassName: topolvm-provisioner
```

#### CloudNativePG (CNPG)

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pg-cluster-1
  namespace: postgresql
  annotations:
    resize.topolvm.io/storage_limit: "500Gi"
    resize.topolvm.io/threshold: "15%"
    resize.topolvm.io/increase: "20Gi"

    resize.topolvm.io/target-resource-class: "cnpg-data"
    resize.topolvm.io/target-resource-name: "my-cluster"
spec:
  accessModes: ["ReadWriteOnce"]
  resources:
    requests:
      storage: 50Gi
  storageClassName: topolvm-provisioner
```

## Helm Configuration

### Full values.yaml Example

```yaml
operatorAwareResizing:
  enabled: true
  resourceClasses:
    # RabbitMQ Operator
    - name: "rabbitmq"
      apiGroup: "rabbitmq.com"
      apiVersion: "v1beta1"
      kind: "RabbitmqCluster"
      resource: "rabbitmqclusters"
      path: "/spec/persistence/storage"

    # CloudNativePG - Data volumes
    - name: "cnpg-data"
      apiGroup: "postgresql.cnpg.io"
      apiVersion: "v1"
      kind: "Cluster"
      resource: "clusters"
      path: "/spec/storage/size"

    # CloudNativePG - WAL volumes
    - name: "cnpg-wal"
      apiGroup: "postgresql.cnpg.io"
      apiVersion: "v1"
      kind: "Cluster"
      resource: "clusters"
      path: "/spec/walStorage/size"

    # Strimzi Kafka
    - name: "kafka"
      apiGroup: "kafka.strimzi.io"
      apiVersion: "v1beta2"
      kind: "Kafka"
      resource: "kafkas"
      path: "/spec/kafka/storage/size"
```

### Deploy or Upgrade

```bash
helm upgrade --install pvc-autoresizer pvc-autoresizer/pvc-autoresizer \
  -n pvc-autoresizer \
  -f values.yaml
```

### Verify Configuration

```bash
# Check RBAC was generated
kubectl get clusterrole pvc-autoresizer-controller -o yaml | grep -A5 "rabbitmq"

# Check ConfigMap was created
kubectl get configmap -n pvc-autoresizer pvc-autoresizer-resource-classes -o yaml
```

## Monitoring

### Metrics

Two Prometheus metrics are available for operator-aware resizing:

#### `pvcautoresizer_cr_patch_success_total`

Counter for successful CR patch operations.

**Labels:**
- `persistentvolumeclaim`: PVC name
- `namespace`: PVC namespace
- `target_kind`: CR kind being patched
- `target_namespace`: CR namespace

#### `pvcautoresizer_cr_patch_failed_total`

Counter for failed CR patch operations.

**Labels:** (same as success metric)

### Events

pvc-autoresizer emits Kubernetes events on the PVC for CR operations:

- **`ResizedCR`** (Normal): Successfully patched the target CR
- **`ResizeCRInvalidConfig`** (Warning): Invalid CR target annotations
- **`ResizeCRFailed`** (Warning): Failed to patch the target CR

## Troubleshooting

### Error: "unknown resource class"

**Symptom**: Event shows `unknown resource class "foo": not defined in controller configuration`

**Cause**: The class name in the PVC annotation doesn't match any class in values.yaml

**Resolution:**
1. Verify the class name in your values.yaml
2. Check for typos in the PVC annotation
3. Run `helm upgrade` if you recently added the class

### Error: "resource classes not configured"

**Symptom**: Event shows `resource classes not configured: cannot use target-resource-class annotation`

**Cause**: `operatorAwareResizing.enabled` is false or `resourceClasses` is empty

**Resolution:**
1. Enable the feature: `operatorAwareResizing.enabled: true`
2. Define at least one resource class
3. Run `helm upgrade`

### Error: "Insufficient permissions to patch CR"

**Symptom**: Event shows `insufficient permissions to patch CR`

**Cause**: RBAC wasn't generated for the CR type

**Resolution:**
1. Ensure the resource class includes the correct `resource` field (plural form)
2. Run `helm upgrade` to regenerate RBAC
3. Verify: `kubectl auth can-i patch <resource>.<apiGroup> --as=system:serviceaccount:pvc-autoresizer:pvc-autoresizer-controller`

### Error: "target CR not found"

**Symptom**: Event shows `target CR not found`

**Cause**: The CR specified in annotations doesn't exist

**Resolution:**
1. Verify the CR exists: `kubectl get <kind> <name> -n <namespace>`
2. Check the `target-resource-name` annotation
3. Verify the CR is in the same namespace as the PVC

## Supported Operators

This feature has been tested and verified with:

- **CloudNativePG (postgresql.cnpg.io/v1)**
  - Data volumes: `.spec.storage.size`
  - WAL volumes: `.spec.walStorage.size`

- **RabbitMQ Operator (rabbitmq.com/v1beta1)**
  - Storage: `.spec.persistence.storage`

This feature should work with any operator that:
- Uses a CR to define storage size
- Reconciles PVC specs based on CR changes
- Accepts Kubernetes Quantity format for storage fields

## Additional Resources

- [Kubernetes Quantity Format](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity/)
- [JSON Pointer (RFC 6901)](https://datatracker.ietf.org/doc/html/rfc6901)
- [Operator Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
