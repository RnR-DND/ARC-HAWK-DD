# Detailed Workflow Documentation

## Overview

This document provides comprehensive, step-by-step workflows for the ARC-Hawk platform, covering Scan Execution, Data Ingestion, Lineage Sync, and Remediation.

---

## 1. System Setup Workflow

### Prerequisites
- Docker & Docker Compose
- Go 1.24+ (Backend)
- Node.js 18+ (Frontend)

### Quick Start (Production)
The recommended way to run the full stack is via Docker Compose:

```bash
# Start all services (Backend, Frontend, DB, Temporal, Neo4j)
docker-compose up -d --build
```

---

## 2. Scan Execution Workflow (Orchestrated)

### Overview
In the Platform mode, scans are orchestrated by **Temporal** to ensure reliability and visibility.

### Step 1: Trigger Scan (API/UI)
User clicks "Start Scan" on Dashboard or calls API:
```http
POST /api/v1/scans/trigger
{
  "profile_name": "production_s3",
  "data_source": "s3",
  "config": { ... }
}
```

### Step 2: Workflow Initialization
Backend starts a Temporal Workflow:
- Workflow ID: `scan-{uuid}`
- Queue: `scan-queue`

### Step 3: Worker Execution
The Scanner Worker picks up the activity:
1.  **Config**: Receives source config and `SCAN_ID`.
2.  **Scan**: Connects to source (S3/DB/etc) and scans content.
3.  **Detect**: Uses spaCy/Regex to find candidates.
4.  **Validate**: Runs Checksum algorithms (Luhn, Verhoeff).
5.  **Ingest**: Calls Backend Ingest API.

---

## 3. Remediation Workflow

### Overview
Automated action to fix findings (e.g., Masking, Deletion).

### Step 1: Request Remediation
User selects findings in "Findings Explorer" and clicks "Remediate".

### Step 2: Execute Action
Backend triggers diverse strategies based on source:
- **Databases**: Executes SQL `UPDATE`.
- **Files/S3**: Downloads, modifies, and re-uploads file.

### Step 3: Verification
The system automatically triggers a **Verification Scan** on the specific asset to confirm the PII is gone.

---

## 4. Lineage Synchronization Workflow

### Overview
Syncs SQL relational data to Neo4j Graph DB.

1.  **Event**: Scan Completion triggers `LineageSyncEvent`.
2.  **Processing**:
    - `Asset` -> Neo4j Node
    - `Findings` -> PII_Category Nodes
    - `Relationships` -> EDGES (`EXPOSES`)
3.  **Result**: Graph is updated immediately for visualization.

---

## 5. Troubleshooting Workflow

### Common Issues

#### Temporal Workflow Stuck
- **Check UI**: `http://localhost:8088`
- **Action**: Terminate workflow and retry.

#### Scanner Connection Refused
- **Cause**: Worker cannot reach Backend API.
- **Fix**: Ensure `API_URL` env var points to the Docker service name (e.g., `http://backend:8080`).

#### Neo4j Sync Failed
- **Cause**: Neo4j container down.
- **Fix**: `docker-compose up -d neo4j`. Backend fails gracefully back to Relational view.
