# ARC-HAWK Findings System Audit Tools

This directory contains comprehensive tools for auditing the integrity and traceability of the ARC-Hawk findings system.

## 📋 Overview

The ARC-Hawk findings system is designed with enterprise-grade data integrity guarantees:
- **Referential Integrity**: Foreign key constraints prevent orphaned records
- **Complete Traceability**: Every finding can be traced to its source
- **Cascade Deletion**: Proper cascade chains maintain data consistency
- **Audit Trails**: Complete history of all data operations

## 🛠️ Audit Tools

### 1. SQL Audit Script (`findings_integrity_audit.sql`)

**Purpose**: Comprehensive SQL-based integrity verification
**Database**: PostgreSQL
**Execution Time**: ~30 seconds for 100K findings

#### What It Checks:

1. **Finding Structure Verification**
   - Required fields presence (scan_run_id, asset_id, pattern_name, severity)
   - Valid PII types (patterns exist in patterns table)
   - Confidence score range validation (0.0-1.0)

2. **Relationship Integrity**
   - Scan-finding relationships (no orphaned findings)
   - Asset-finding relationships (no findings without assets)
   - Statistics accuracy (reported vs actual counts)

3. **Location Traceability**
   - Exact file paths preserved
   - Asset deduplication via stable IDs
   - No duplicate locations without proper identification

4. **Validation Logic References**
   - Pattern references are valid
   - Classifications exist for all findings
   - Validation methods are properly recorded

5. **Aggregation Integrity**
   - Summary statistics match individual records
   - No hidden aggregations
   - Proper filtering of Non-PII findings

6. **Cascade Deletion Behavior**
   - Foreign key constraints properly configured
   - No circular dependencies
   - Proper cascade chains (scan→findings→classifications)

7. **Complete Traceability**
   - End-to-end audit trail verification
   - Finding→Asset→Scan→Classification chain integrity

#### Usage:

```bash
# Run directly with psql
psql -h localhost -U postgres -d arc_platform -f findings_integrity_audit.sql

# Or use the audit runner script
./run_audit.sh
```

### 2. Go Validation Service (`findings_validation.go`)

**Purpose**: Programmatic integrity verification with detailed reporting
**Language**: Go
**Execution Time**: ~15 seconds for 100K findings

#### Features:

- JSON output for CI/CD integration
- Detailed error reporting and recommendations
- Configurable timeouts and retry logic
- Exit codes for automated testing

#### Usage:

```bash
# Set database connection
export DATABASE_URL="postgres://postgres:password123@localhost:5432/arc_platform?sslmode=disable"

# Build and run
go build findings_validation.go
./findings_validation

# Or run with Go directly
go run findings_validation.go
```

#### Exit Codes:
- `0`: All checks passed
- `1`: Critical issues found
- `2`: Failures found
- `3`: Warnings only

### 3. Audit Runner Script (`run_audit.sh`)

**Purpose**: Easy-to-use wrapper for running audits
**Language**: Bash
**Features**: Database connection testing, formatted output, exit codes

#### Usage:

```bash
# Run with default settings
./run_audit.sh

# Or with custom database settings
DB_HOST=myhost DB_PORT=5432 DB_NAME=arc_platform DB_USER=myuser ./run_audit.sh
```

## 📊 Audit Results Interpretation

### Status Levels

| Status | Meaning | Action Required |
|--------|---------|-----------------|
| **PASS** | ✅ No issues found | None |
| **WARNING** | ⚠️ Minor issues identified | Review and address if time permits |
| **FAIL** | ❌ Significant issues | Must fix before production |
| **CRITICAL** | 🚨 Serious integrity problems | Immediate attention required |

### Key Metrics

The audit generates the following key metrics:

1. **Data Integrity Score**: Percentage of findings with complete, valid data
2. **Traceability Score**: Percentage of findings with complete audit trails
3. **Relationship Integrity**: Verification of all foreign key relationships
4. **Cascade Safety**: Verification of proper deletion behavior

## 🚨 Common Issues and Solutions

### Issue: Orphaned Findings
**Symptoms**: Findings without valid scan_run_id or asset_id
**Causes**: Manual database modifications, failed transactions
**Solution**: Identify and clean up orphaned records, restore missing parent records

```sql
-- Find orphaned findings
SELECT f.id, f.scan_run_id, f.asset_id 
FROM findings f 
LEFT JOIN scan_runs sr ON f.scan_run_id = sr.id 
LEFT JOIN assets a ON f.asset_id = a.id 
WHERE sr.id IS NULL OR a.id IS NULL;
```

### Issue: Statistics Inconsistency
**Symptoms**: total_findings doesn't match actual finding count
**Causes**: Timing issues during scan ingestion, manual updates
**Solution**: Run statistics refresh script

```sql
-- Refresh scan run statistics
UPDATE scan_runs sr 
SET total_findings = (
    SELECT COUNT(*) 
    FROM findings f 
    WHERE f.scan_run_id = sr.id
);
```

### Issue: Missing Classifications
**Symptoms**: Findings without classification records
**Causes**: Failed classification during ingestion
**Solution**: Re-run classification process

```sql
-- Find findings without classifications
SELECT f.id, f.pattern_name 
FROM findings f 
LEFT JOIN classifications c ON f.id = c.finding_id 
WHERE c.finding_id IS NULL;
```

## 🔄 CI/CD Integration

### GitHub Actions Example

```yaml
name: Findings Integrity Audit
on: [push, pull_request]

jobs:
  audit:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: password123
          POSTGRES_DB: arc_platform
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Setup Go
      uses: actions/setup-go@v3
      with:
        go-version: '1.21'
    
    - name: Run Integrity Audit
      env:
        DATABASE_URL: postgres://postgres:password123@localhost:5432/arc_platform?sslmode=disable
      run: |
        cd scripts/audit
        go run findings_validation.go
```

## 📈 Performance Considerations

### Large Dataset Auditing
For datasets > 1M findings, consider:

1. **Parallel Processing**: Use database partitions for parallel checks
2. **Sampling**: Audit random 10% sample for regular monitoring
3. **Incremental Auditing**: Only audit newly added/modified records

### Optimization Tips

```sql
-- Add indexes for faster audit queries
CREATE INDEX CONCURRENTLY idx_audit_finding_scan_asset ON findings(scan_run_id, asset_id);
CREATE INDEX CONCURRENTLY idx_audit_pattern_name ON findings(pattern_name);
```

## 🛡️ Security Considerations

### Database Access
- Use read-only database user for audits
- Encrypt database connections in production
- Limit audit tool access to authorized personnel

### Data Privacy
- Audit reports may contain PII references
- Store audit results securely
- Consider redacting sample_text in production audit reports

## 📞 Support

### Running into Issues?
1. Check database connectivity first
2. Verify database schema is up-to-date
3. Review audit logs for specific error messages
4. Check this README for common solutions

### Getting Help
- Review the comprehensive audit report: `AUDIT_REPORT.md`
- Check the main project documentation: `../../AGENTS.md`
- Review the database schema: `../../apps/backend/migrations_versioned/`

---

## 📝 Audit History

| Date | Version | Changes |
|------|---------|---------|
| 2026-01-23 | 1.0.0 | Initial comprehensive audit suite |
| | | Added SQL and Go validation tools |
| | | Included CI/CD integration examples |
| | | Documented common issues and solutions |

---

**Note**: These audit tools are designed specifically for the ARC-Hawk findings system architecture. They leverage the specific database schema, entity relationships, and business logic implemented in the platform.
