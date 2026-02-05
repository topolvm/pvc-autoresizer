# PVC Autoresizer Examples

This directory contains practical examples for using pvc-autoresizer with operator-aware resizing.

## Files

### `annotate-pvcs.sh`

Shell script with kubectl commands to manually annotate PVCs for different operators:
- CloudNativePG (data, WAL, and tablespace volumes)
- RabbitMQ Operator
- Strimzi Kafka Operator

**Usage:**
```bash
# View the examples
cat annotate-pvcs.sh

# Copy and modify commands for your environment
# Replace cluster names, namespaces, and limits as needed
```

### `kyverno-cnpg-autoresizer.yaml`

Kyverno ClusterPolicy that automatically adds pvc-autoresizer annotations to CloudNativePG PVCs.

**Features:**
- Automatically detects PVC type (data, WAL, or tablespace) using `cnpg.io/pvcRole` label
- Applies correct resource class for each volume type:
  - `PG_DATA` → `cnpg-data`
  - `PG_WAL` → `cnpg-wal`
  - `PG_TABLESPACE` → `cnpg-tablespace` (with `target-filter-value` set to tablespace name)
- Extracts cluster name from `cnpg.io/cluster` label
- Includes production-tuned variant with namespace selector

**Installation:**
```bash
# Install Kyverno (if not already installed)
# See https://kyverno.io/docs/installation/ for installation options
kubectl create -f https://github.com/kyverno/kyverno/releases/download/v1.12.0/install.yaml

# Apply the policy
kubectl apply -f examples/kyverno-cnpg-autoresizer.yaml

# Verify policy is ready
kubectl get clusterpolicy add-cnpg-pvc-autoresizer-annotations

# Create a CNPG cluster - PVCs will automatically get annotations
kubectl apply -f - <<EOF
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: my-postgres
  namespace: database
spec:
  instances: 3
  storage:
    size: 50Gi
    storageClass: default
  walStorage:
    size: 10Gi
    storageClass: default
EOF

# Verify annotations were added
kubectl get pvc my-postgres-1 -n database -o yaml | grep resize.topolvm.io
kubectl get pvc my-postgres-1-wal -n database -o yaml | grep resize.topolvm.io
```

**Customization:**

Edit the policy to adjust default values:
- `storage_limit`: Maximum PVC size
- `threshold`: Percentage of free space to trigger resize
- `increase`: Amount to increase by

For production environments, use the `-prod` variant which includes namespace selectors.

## Testing

### Test CloudNativePG with Separate WAL Volumes

```bash
# 1. Apply Kyverno policy
kubectl apply -f examples/kyverno-cnpg-autoresizer.yaml

# 2. Create test namespace
kubectl create namespace database

# 3. Create CNPG cluster with WAL volumes
kubectl apply -f - <<EOF
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: test-cluster
  namespace: database
spec:
  instances: 1
  storage:
    size: 1Gi
    storageClass: default
  walStorage:
    size: 1Gi
    storageClass: default
EOF

# 4. Wait for PVCs to be created
kubectl wait --for=condition=ready pod -l cnpg.io/cluster=test-cluster -n database --timeout=120s

# 5. Verify annotations were auto-applied
kubectl get pvc -n database -l cnpg.io/cluster=test-cluster -o yaml | grep -A 8 "resize.topolvm.io"

# 6. Fill disk to trigger resize (>80% full)
kubectl exec -n database test-cluster-1 -- \
  dd if=/dev/zero of=/var/lib/postgresql/data/pgdata/fill bs=1M count=800

# 7. Monitor for resize events
kubectl get events -n database --watch | grep -i resize

# 8. Check CR was updated
kubectl get cluster test-cluster -n database -o jsonpath='{.spec.storage.size}'
kubectl get cluster test-cluster -n database -o jsonpath='{.spec.walStorage.size}'
```

### Test CloudNativePG with Tablespaces

Tablespace resizing is fully supported and tested:

```bash
# 1. Apply Kyverno policy
kubectl apply -f examples/kyverno-cnpg-autoresizer.yaml

# 2. Create test namespace
kubectl create namespace database

# 3. Create CNPG cluster with tablespaces
kubectl apply -f - <<EOF
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: test-pg-tablespace
  namespace: database
spec:
  instances: 1
  storage:
    size: 1Gi
    storageClass: default
  walStorage:
    size: 1Gi
    storageClass: default
  tablespaces:
    - name: mydata
      storage:
        size: 1Gi
        storageClass: default
    - name: myindex
      storage:
        size: 1Gi
        storageClass: default
EOF

# 4. Wait for cluster to be ready
kubectl wait --for=condition=ready pod -l cnpg.io/cluster=test-pg-tablespace -n database --timeout=180s

# 5. Verify Kyverno applied annotations to tablespace PVCs
kubectl get pvc -n database -l cnpg.io/pvcRole=PG_TABLESPACE -o yaml | grep -A 10 "resize.topolvm.io"

# Expected output should show:
# resize.topolvm.io/target-resource-class: cnpg-tablespace
# resize.topolvm.io/target-filter-value: mydata

# 6. Fill tablespace to trigger resize
kubectl exec -n database test-pg-tablespace-1 -- \
  psql -U postgres -c "CREATE TABLE mydata.test_table AS SELECT generate_series(1,100000) as id, md5(random()::text) as data TABLESPACE mydata;"

# Or use dd to fill the filesystem directly:
kubectl exec -n database test-pg-tablespace-1 -- \
  dd if=/dev/zero of=/var/lib/postgresql/tablespaces/mydata/fill bs=1M count=800

# 7. Monitor for resize events (may take up to 6 minutes for metrics to update)
kubectl get events -n database --watch | grep -i resize

# Expected events:
# - ResizedCR: pvc-autoresizer patches the CR
# - Resizing/ExternalExpanding: CSI driver starts resize
# - FileSystemResizeRequired: Volume expanded, filesystem resize pending
# - FileSystemResizeSuccessful: Filesystem expanded

# 8. Verify resize completed
kubectl get cluster test-pg-tablespace -n database -o jsonpath='{.spec.tablespaces[?(@.name=="mydata")].storage.size}'
kubectl exec -n database test-pg-tablespace-1 -- df -h /var/lib/postgresql/tablespaces/mydata

# Should show expanded size (e.g., 2.0G if threshold was 20% and increase was 1Gi)
```

**Expected Timeline:**
- **T+0**: Fill tablespace to >80% usage
- **T+6m**: pvc-autoresizer detects threshold breach, patches CR
- **T+6m**: CNPG operator updates PVC spec
- **T+7m**: CSI driver resizes underlying volume
- **T+12m**: Kubelet expands filesystem

**Verified Behavior:**
- ✅ Kyverno automatically adds correct JSONPath for tablespace PVCs
- ✅ pvc-autoresizer successfully patches CR tablespace size
- ✅ CNPG operator updates PVC spec to match CR
- ✅ CSI driver and kubelet complete volume expansion
- ✅ Filesystem expanded to new size

## Advanced Scenarios

### Tablespace Support

For PostgreSQL clusters with tablespaces:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: my-postgres
  namespace: database
spec:
  instances: 3
  storage:
    size: 100Gi
  tablespaces:
    - name: mydata
      storage:
        size: 500Gi
        storageClass: fast-ssd
```

The Kyverno policy will automatically add annotations to the tablespace PVC with:
```
resize.topolvm.io/target-resource-class: cnpg-tablespace
resize.topolvm.io/target-filter-value: mydata
```

Each tablespace gets its own PVC with the `cnpg-tablespace` class and the tablespace name as the filter value.

## Troubleshooting

### Policy Not Applying Annotations

```bash
# Check policy status
kubectl get clusterpolicy add-cnpg-pvc-autoresizer-annotations -o yaml

# Check for events
kubectl get events -n kyverno --field-selector reason=PolicyApplied

# Verify PVC has correct labels
kubectl get pvc <pvc-name> -n <namespace> -o jsonpath='{.metadata.labels}'
```

### Annotations Not Working

```bash
# Verify pvc-autoresizer sees the annotations
kubectl logs -n pvc-autoresizer deployment/pvc-autoresizer-controller | grep <pvc-name>

# Check for validation errors
kubectl get events -n <namespace> | grep <pvc-name>

# Verify JSON path is correct
kubectl get cluster <cluster-name> -n <namespace> -o yaml | yq eval '.spec.storage.size'
```

## References

- [CloudNativePG Documentation](https://cloudnative-pg.io/)
- [CloudNativePG Discussion #2321](https://github.com/cloudnative-pg/cloudnative-pg/discussions/2321)
- [Kyverno Documentation](https://kyverno.io/)
- [pvc-autoresizer Operator-Aware Resizing Docs](../docs/operator-aware-resizing.md)
