# Failure Modes & Recovery

## 1. Temporal Unavailable
**Impact**: Critical. Scans and Remediation cannot start.
**Symptoms**:
- "Failed to start workflow" errors in API.
- Dashboard shows "System Unhealthy".
**Recovery**:
- Restart Temporal: `docker-compose restart temporal`
- Check logs: `docker-compose logs temporal`

## 2. Neo4j Unavailable
**Impact**: Medium. Lineage graph functionality disabled.
**Symptoms**:
- Lineage page shows empty or error.
- Core scanning and findings still work (Graceful Degradation).
**Recovery**:
- Restart Neo4j: `docker-compose up -d neo4j`

## 3. Scanner Worker Crash
**Impact**: Low. Individual scan job fails, but system remains up.
**Symptoms**:
- Scan stuck in "Running" state.
**Recovery**:
- Temporal automatically retries the workflow.
- If stuck > 1 hour, cancel via Temporal UI (`localhost:8088`).

## 4. Database Connection Lost
**Impact**: Critical. System Inoperable.
**Symptoms**:
- API 500 Errors.
**Recovery**:
- Verify Postgres container is running.
- Check disk space.
