#!/bin/bash
# Examples of adding operator-aware resizing annotations to PVCs
# These commands have been tested and verified to work correctly

# =============================================================================
# CloudNativePG (CNPG) PostgreSQL Cluster
# =============================================================================
# For a CNPG Cluster named "my-postgres" in namespace "database"
# with a PVC named "my-postgres-1"

kubectl annotate pvc my-postgres-1 -n database \
  resize.topolvm.io/enabled="true" \
  resize.topolvm.io/storage_limit="500Gi" \
  resize.topolvm.io/threshold="20%" \
  resize.topolvm.io/increase="10Gi" \
  resize.topolvm.io/target-resource-api-version="postgresql.cnpg.io/v1" \
  resize.topolvm.io/target-resource-kind="Cluster" \
  resize.topolvm.io/target-resource-name="my-postgres" \
  resize.topolvm.io/target-resource-json-path=".spec.storage.size"

# Note: If the Cluster is in a different namespace, add:
#   resize.topolvm.io/target-resource-namespace="other-namespace"

# -----------------------------------------------------------------------------
# CloudNativePG with Separate WAL Storage
# -----------------------------------------------------------------------------
# For a CNPG Cluster with separate WAL volumes
# WAL PVC is named "my-postgres-1-wal"

kubectl annotate pvc my-postgres-1-wal -n database \
  resize.topolvm.io/enabled="true" \
  resize.topolvm.io/storage_limit="200Gi" \
  resize.topolvm.io/threshold="20%" \
  resize.topolvm.io/increase="10Gi" \
  resize.topolvm.io/target-resource-api-version="postgresql.cnpg.io/v1" \
  resize.topolvm.io/target-resource-kind="Cluster" \
  resize.topolvm.io/target-resource-name="my-postgres" \
  resize.topolvm.io/target-resource-json-path=".spec.walStorage.size"

# -----------------------------------------------------------------------------
# CloudNativePG with Tablespaces
# -----------------------------------------------------------------------------
# For a CNPG Cluster with tablespaces
# Tablespace PVC is named "my-postgres-1-mydata" (for tablespace named "mydata")

kubectl annotate pvc my-postgres-1-mydata -n database \
  resize.topolvm.io/enabled="true" \
  resize.topolvm.io/storage_limit="1000Gi" \
  resize.topolvm.io/threshold="20%" \
  resize.topolvm.io/increase="50Gi" \
  resize.topolvm.io/target-resource-api-version="postgresql.cnpg.io/v1" \
  resize.topolvm.io/target-resource-kind="Cluster" \
  resize.topolvm.io/target-resource-name="my-postgres" \
  resize.topolvm.io/target-resource-json-path=".spec.tablespaces[?(@.name=='mydata')].storage.size"

# Note: Replace 'mydata' with your actual tablespace name in both the PVC name
# and the JSON path filter

# =============================================================================
# RabbitMQ Operator
# =============================================================================
# For a RabbitmqCluster named "my-rabbitmq" in namespace "messaging"
# with a PVC named "persistence-my-rabbitmq-server-0"

kubectl annotate pvc persistence-my-rabbitmq-server-0 -n messaging \
  resize.topolvm.io/enabled="true" \
  resize.topolvm.io/storage_limit="100Gi" \
  resize.topolvm.io/threshold="20%" \
  resize.topolvm.io/increase="10Gi" \
  resize.topolvm.io/target-resource-api-version="rabbitmq.com/v1beta1" \
  resize.topolvm.io/target-resource-kind="RabbitmqCluster" \
  resize.topolvm.io/target-resource-name="my-rabbitmq" \
  resize.topolvm.io/target-resource-json-path=".spec.persistence.storage"

# =============================================================================
# Strimzi Kafka Operator
# =============================================================================
# For a Kafka cluster named "my-kafka-cluster" in namespace "kafka"
# with a PVC named "data-my-kafka-cluster-kafka-0"

kubectl annotate pvc data-my-kafka-cluster-kafka-0 -n kafka \
  resize.topolvm.io/enabled="true" \
  resize.topolvm.io/storage_limit="1000Gi" \
  resize.topolvm.io/threshold="20%" \
  resize.topolvm.io/increase="50Gi" \
  resize.topolvm.io/target-resource-api-version="kafka.strimzi.io/v1beta2" \
  resize.topolvm.io/target-resource-kind="Kafka" \
  resize.topolvm.io/target-resource-name="my-kafka-cluster" \
  resize.topolvm.io/target-resource-json-path=".spec.kafka.storage.size"

# For ZooKeeper PVCs (data-my-kafka-cluster-zookeeper-0), use:
#   resize.topolvm.io/target-resource-json-path=".spec.zookeeper.storage.size"

# =============================================================================
# Updating Existing Annotations
# =============================================================================
# If annotations already exist, add --overwrite flag:

kubectl annotate pvc my-pvc -n my-namespace --overwrite \
  resize.topolvm.io/storage_limit="1000Gi"

# =============================================================================
# Verifying Annotations
# =============================================================================
# Check all resize annotations on a PVC:

kubectl get pvc my-pvc -n my-namespace -o jsonpath='{.metadata.annotations}' | jq 'with_entries(select(.key | startswith("resize.topolvm.io")))'

# Or view specific annotation:
kubectl get pvc my-pvc -n my-namespace -o jsonpath='{.metadata.annotations.resize\.topolvm\.io/target-resource-name}'
