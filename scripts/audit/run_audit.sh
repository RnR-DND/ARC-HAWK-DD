#!/bin/bash

# ARC-HAWK Findings Integrity Audit Runner
# This script runs comprehensive integrity checks on the findings system

set -e

echo "🔍 ARC-HAWK Findings System Integrity Audit"
echo "=========================================="
echo "Date: $(date)"
echo "Auditor: Automated Integrity Check"
echo ""

# Check if PostgreSQL is available
if ! command -v psql &> /dev/null; then
    echo "❌ ERROR: psql command not found. Please install PostgreSQL client."
    exit 1
fi

# Database connection parameters
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${DB_NAME:-arc_platform}
DB_USER=${DB_USER:-postgres}
DB_PASSWORD=${DB_PASSWORD:-password123}

# Set PGPASSWORD for non-interactive authentication
export PGPASSWORD=$DB_PASSWORD

# Test database connection
echo "📡 Testing database connection..."
if ! psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT 1;" &> /dev/null; then
    echo "❌ ERROR: Cannot connect to database. Please check connection parameters."
    echo "   Host: $DB_HOST"
    echo "   Port: $DB_PORT" 
    echo "   Database: $DB_NAME"
    echo "   User: $DB_USER"
    exit 1
fi
echo "✅ Database connection successful"
echo ""

# Run SQL audit
echo "📊 Running SQL integrity audit..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$(dirname "$0")/findings_integrity_audit.sql"

echo ""
echo "📋 Generating summary report..."

# Get audit summary
echo "=== AUDIT SUMMARY ==="
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
SELECT 
    status,
    COUNT(*) as test_count,
    STRING_AGG(test_name, ', ') as tests
FROM audit_results 
GROUP BY status 
ORDER BY 
    CASE status 
        WHEN 'CRITICAL' THEN 1 
        WHEN 'FAIL' THEN 2 
        WHEN 'WARNING' THEN 3 
        WHEN 'PASS' THEN 4 
    END;
"

echo ""
echo "🎯 System Health Check:"

# Check for critical issues
critical_count=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM audit_results WHERE status = 'CRITICAL';" | tr -d ' ')

if [ "$critical_count" -gt 0 ]; then
    echo "🚨 CRITICAL ISSUES FOUND: $critical_count"
    echo "   Immediate attention required!"
    exit 1
fi

# Check for failures
fail_count=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM audit_results WHERE status = 'FAIL';" | tr -d ' ')

if [ "$fail_count" -gt 0 ]; then
    echo "❌ FAILURES FOUND: $fail_count"
    echo "   Issues must be addressed before production deployment."
    exit 2
fi

# Check for warnings
warning_count=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM audit_results WHERE status = 'WARNING';" | tr -d ' ')

if [ "$warning_count" -gt 0 ]; then
    echo "⚠️  WARNINGS FOUND: $warning_count"
    echo "   Should be reviewed but not blocking."
fi

# Check for passes
pass_count=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM audit_results WHERE status = 'PASS';" | tr -d ' ')

echo "✅ PASSED TESTS: $pass_count"

# Get total findings count
total_findings=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM findings;" | tr -d ' ')
echo "📈 Total Findings in System: $total_findings"

echo ""
echo "🎉 AUDIT COMPLETED SUCCESSFULLY"
echo "================================"

if [ "$warning_count" -gt 0 ]; then
    echo "Status: PASSED with warnings"
    exit 3
else
    echo "Status: PASSED - All checks successful"
    exit 0
fi
