# Agent 4: Database Infrastructure -- COMPLETE

## Summary

All database infrastructure for ARL-Hawk has been built across three directories under `hawk/`.

## Deliverables

### SQL Migrations (`hawk/migrations/`)

| File | Contents |
|------|----------|
| `000001_initial_schema.up.sql` | Core tables: schema_migrations, sources, scan_jobs, assets (with GIN full-text search), findings (date-partitioned monthly 2026 H1), risk_scores, remediation_issues, audit_log |
| `000001_initial_schema.down.sql` | Drops all core tables in dependency order |
| `000002_profiles_policies.up.sql` | profiles (Keycloak-linked), policies (RBAC), profile_policy_assignments (M:N join) |
| `000002_profiles_policies.down.sql` | Drops profile/policy tables |
| `000003_custom_regex_patterns.up.sql` | custom_regex_patterns (soft-delete, auto-deactivation at 30% FP rate), custom_regex_match_log |
| `000003_custom_regex_patterns.down.sql` | Drops regex tables |
| `000004_agent_sync_log.up.sql` | agent_sync_log with composite PK (agent_id, scan_job_id, batch_seq) for idempotent offline sync |
| `000004_agent_sync_log.down.sql` | Drops agent sync table |
| `000005_citus_distribution.up.sql` | Distributes assets/findings/risk_scores/scan_jobs; references sources/profiles/policies/custom_regex_patterns |
| `000005_citus_distribution.down.sql` | Undistributes all tables |
| `000006_consent_records.up.sql` | DPDPA Sec 4/6 consent_records with generated is_valid column |
| `000006_consent_records.down.sql` | Drops consent table |

All migrations are wrapped in transactions, sequentially numbered, and fully reversible.

### Kubernetes -- PostgreSQL + Citus (`hawk/k8s/postgres/`)

| File | Purpose |
|------|---------|
| `statefulset.yaml` | Citus coordinator (1 replica, 50Gi PVC, liveness/readiness probes, anti-affinity) |
| `workers-statefulset.yaml` | Citus workers (2 replicas, 100Gi PVC each, init container waits for coordinator) |
| `pgbouncer-deployment.yaml` | PgBouncer (2 replicas, transaction mode, 400 max connections) |
| `services.yaml` | hawk-postgres-write (coordinator), hawk-postgres-read (PgBouncer), headless services |
| `pvc.yaml` | PVC templates for manual provisioning |
| `configmap.yaml` | PgBouncer config (pgbouncer.ini, userlist.txt) + Postgres init scripts (extensions, worker registration) |
| `secret.yaml` | Helm-templated secret (POSTGRES_USER, POSTGRES_PASSWORD, DATABASE_URL, PGBOUNCER_URL) |
| `pdb.yaml` | PDB for workers (minAvailable: 2) and PgBouncer (minAvailable: 1) |

### Kubernetes -- Neo4j Causal Cluster (`hawk/k8s/neo4j/`)

| File | Purpose |
|------|---------|
| `statefulset.yaml` | 3-node causal cluster (50Gi each, discovery/raft/transaction ports, anti-affinity) |
| `haproxy-configmap.yaml` | HAProxy config for bolt + HTTP read routing with health checks |
| `haproxy-deployment.yaml` | HAProxy deployment (2 replicas, stats endpoint on 8404) |
| `services.yaml` | hawk-neo4j-write, hawk-neo4j-read (via HAProxy), hawk-neo4j-headless |
| `pvc.yaml` | PVC templates for all 3 nodes |
| `fabric-configmap.yaml` | Fabric tenant routing config + Neo4j tuning (memory, query logging, timeouts) |
| `pdb.yaml` | PDB for Neo4j (minAvailable: 2) and HAProxy (minAvailable: 1) |

### Kubernetes -- Redis Cluster (`hawk/k8s/redis/`)

| File | Purpose |
|------|---------|
| `statefulset.yaml` | 6-node cluster (3 primary + 3 replica, 10Gi each, sysctl init container, anti-affinity) |
| `services.yaml` | hawk-redis (client) + hawk-redis-headless (gossip) |
| `configmap.yaml` | Cluster config (AOF persistence, 768mb maxmemory, allkeys-lru eviction, pub/sub buffers) |
| `pdb.yaml` | PDB (minAvailable: 4 of 6 to preserve quorum) |

## Design Decisions

- **All secrets are templates only** -- values populated by Helm at deploy time. No real credentials committed.
- **All StatefulSets have podAntiAffinity** spreading pods across K8s nodes.
- **All containers have liveness + readiness probes** with appropriate initial delays.
- **All containers have resource requests and limits** sized per workload.
- **PodDisruptionBudgets** on all stateful workloads to prevent disruption below quorum.
- **Findings table is date-partitioned** (monthly, 2026 H1) for scan result volume.
- **Citus distribution** shards high-volume tables by source_id/scan_job_id, replicates small reference tables.
- **consent_records.is_valid** is a generated column for efficient DPDPA compliance queries.

## File Count

- 12 SQL migration files (6 up + 6 down)
- 8 Postgres K8s manifests
- 7 Neo4j K8s manifests
- 4 Redis K8s manifests
- **Total: 31 files + this summary = 32 files**
