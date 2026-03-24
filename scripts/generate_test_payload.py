#!/usr/bin/env python3
"""
Generate test payload for SDK-verified ingestion endpoint
Tests POST /api/v1/scans/ingest-verified
"""
import json
import hashlib
from datetime import datetime

def generate_test_payload():
    """Generate realistic SDK-verified findings"""
    
    findings = [
        {
            "pii_type": "AADHAAR",
            "value_hash": hashlib.sha256(b"234123412346").hexdigest(),
            "source": {
                "asset_name": "customer_data.csv",
                "asset_path": "/data/customers/customer_data.csv",
                "asset_type": "file",
                "line": 42
            },
            "validators_passed": ["verhoeff", "dummy_detector"],
            "ml_confidence": 0.95,
            "context_excerpt": "Customer ID: 12345, Aadhaar: 2341-2341-2346, Status: Active",
            "context_keywords": ["customer", "aadhaar", "verification"],
            "sdk_version": "2.0",
            "metadata": {
                "scan_timestamp": str(datetime.now().isoformat())
            }
        },
        {
            "pii_type": "PAN",
            "value_hash": hashlib.sha256(b"ABCDE1234F").hexdigest(),
            "source": {
                "asset_name": "tax_records",
                "asset_path": "/data/finance/tax_records",
                "asset_type": "database",
                "column": "pan_number",
                "table_name": "employees"
            },
            "validators_passed": ["format_check"],
            "ml_confidence": 0.98,
            "context_excerpt": "Employee PAN: ABCDE1234F for tax filing",
            "context_keywords": ["pan", "tax", "employee"],
            "sdk_version": "2.0",
            "metadata": {}
        },
        {
            "pii_type": "CREDIT_CARD",
            "value_hash": hashlib.sha256(b"4532015112830366").hexdigest(),
            "source": {
                "asset_name": "payments.db",
                "asset_path": "/data/payments/payments.db",
                "asset_type": "database",
                "column": "card_number",
                "table_name": "transactions"
            },
            "validators_passed": ["luhn"],
            "ml_confidence": 1.0,
            "context_excerpt": "Payment processed via card ****0366",
            "context_keywords": ["payment", "card", "transaction"],
            "sdk_version": "2.0",
            "metadata": {}
        }
    ]
    
    payload = {
        "scan_id": f"sdk-test-{int(datetime.now().timestamp())}",
        "findings": findings,
        "metadata": {
            "scanner": "unified-scan-sdk.py",
            "timestamp": datetime.now().isoformat(),
            "total_files_scanned": 3
        }
    }
    
    return payload

if __name__ == "__main__":
    payload = generate_test_payload()
    
    # Write to file
    with open("test_sdk_payload.json", "w") as f:
        json.dump(payload, f, indent=2)
    
    print("✓ Created test_sdk_payload.json")
    print(f"  - {len(payload['findings'])} findings")
    print(f"  - Scan ID: {payload['scan_id']}")
    print("\nTo test:")
    print("  curl -X POST http://localhost:8080/api/v1/scans/ingest-verified \\")
    print("    -H 'Content-Type: application/json' \\")
    print("    -d @test_sdk_payload.json")
