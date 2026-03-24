#!/usr/bin/env python3
"""
Test payload for SDK-verified ingestion endpoint
Simulates output from unified-scan-sdk.py
"""

import json
import hashlib
from datetime import datetime

def create_test_payload():
    """Create a test VerifiedScanInput payload."""
    
    # Sample verified findings
    findings = [
        {
            "pii_type": "IN_AADHAAR",
            "value_hash": hashlib.sha256(b"999911112226").hexdigest(),
            "source": {
                "path": "/test/data/users.csv",
                "line": 42,
                "column": "aadhaar_number",
                "data_source": "filesystem",
                "host": "test-machine"
            },
            "validators_passed": ["verhoeff"],
            "validation_method": "mathematical",
            "ml_confidence": 0.91,
            "ml_entity_type": "IN_AADHAAR",
            "context_excerpt": "Customer Aadhaar 9999 1111 2226 enrolled",
            "context_keywords": ["aadhaar", "customer"],
            "pattern_name": "Aadhaar",
            "detected_at": datetime.utcnow().isoformat() + "Z",
            "scanner_version": "2.0-sdk"
        },
        {
            "pii_type": "IN_PAN",
            "value_hash": hashlib.sha256(b"ABCDE1234F").hexdigest(),
            "source": {
                "path": "/test/data/tax_records.csv",
                "line": 15,
                "column": "pan_number",
                "data_source": "filesystem",
                "host": "test-machine"
            },
            "validators_passed": ["format"],
            "validation_method": "format",
            "ml_confidence": 0.85,
            "ml_entity_type": "IN_PAN",
            "context_excerpt": "PAN ABCDE1234F for tax filing",
            "context_keywords": ["pan", "tax"],
            "pattern_name": "PAN",
            "detected_at": datetime.utcnow().isoformat() + "Z",
            "scanner_version": "2.0-sdk"
        },
        {
            "pii_type": "CREDIT_CARD",
            "value_hash": hashlib.sha256(b"4532015112830366").hexdigest(),
            "source": {
                "path": "/test/data/payments.db",
                "table": "transactions",
                "column": "card_number",
                "data_source": "postgresql",
                "host": "localhost"
            },
            "validators_passed": ["luhn"],
            "validation_method": "mathematical",
            "ml_confidence": 0.95,
            "ml_entity_type": "CREDIT_CARD",
            "context_excerpt": "Card 4532 0151 1283 0366 on file",
            "context_keywords": ["card", "payment"],
            "pattern_name": "Credit_Card",
            "detected_at": datetime.utcnow().isoformat() + "Z",
            "scanner_version": "2.0-sdk"
        }
    ]
    
    payload = {
        "verified_findings": findings,
        "scanner_version": "2.0-sdk",
        "validation_enabled": True
    }
    
    return payload


if __name__ == "__main__":
    payload = create_test_payload()
    
    # Save to file
    output_file = "test_verified_payload.json"
    with open(output_file, 'w') as f:
        json.dump(payload, f, indent=2)
    
    print(f"✓ Created test payload: {output_file}")
    print(f"  - {len(payload['verified_findings'])} verified findings")
    print(f"  - Scanner version: {payload['scanner_version']}")
    print(f"\nTo test:")
    print(f"  curl -X POST http://localhost:8080/api/v1/scans/ingest-verified \\")
    print(f"    -H 'Content-Type: application/json' \\")
    print(f"    -d @{output_file}")
