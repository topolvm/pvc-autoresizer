#!/bin/bash
# Examples of adding operator-aware resizing annotations to PVCs
# These commands use admin-defined resource classes for security

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
  resize.topolvm.io/target-resource-class="cnpg-data" \
  resize.topolvm.io/target-resource-name="my-postgres"

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
  resize.topolvm.io/target-resource-class="cnpg-wal" \
  resize.topolvm.io/target-resource-name="my-postgres"

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
  resize.topolvm.io/target-resource-class="cnpg-tablespace" \
  resize.topolvm.io/target-resource-name="my-postgres" \
  resize.topolvm.io/target-filter-value="mydata"

# Note: Replace 'mydata' with your actual tablespace name in both the PVC name
# and the target-filter-value annotation

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
  resize.topolvm.io/target-resource-class="rabbitmq" \
  resize.topolvm.io/target-resource-name="my-rabbitmq"

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
  resize.topolvm.io/target-resource-class="kafka" \
  resize.topolvm.io/target-resource-name="my-kafka-cluster"

# For ZooKeeper PVCs (data-my-kafka-cluster-zookeeper-0), use:
#   resize.topolvm.io/target-resource-class="kafka-zookeeper"

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
kubectl get pvc my-pvc -n my-namespace -o jsonpath='{.metadata.annotations.resize\.topolvm\.io/target-resource-class}'
