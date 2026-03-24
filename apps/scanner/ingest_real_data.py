#!/usr/bin/env python3
"""
Direct ingestion of real scan results to ARC-Hawk backend
"""

import requests
import json
import time
from datetime import datetime

def ingest_real_scan_results():
    """Send real PII findings to backend"""
    
    # Read our real scan results
    with open('real_scan_results_manual.json', 'r') as f:
        scan_data = json.load(f)
    
    # Backend ingest URL
    ingest_url = "http://localhost:8080/api/v1/scans/ingest-verified"
    
    # Prepare findings in backend format
    findings = []
    
    # Process Aadhaar findings
    if 'fs' in scan_data:
        for result in scan_data['fs']:
            if result['pattern_name'] == 'IN_AADHAAR':
                for match in result['matches']:
                    findings.append({
                        "pattern_name": "IN_AADHAAR",
                        "matches": [match],  # Backend expects list
                        "sample_text": result['sample_text'],
                        "confidence_score": result['validation_method']['confidence'],
                        "validation_method": json.dumps(result['validation_method']),
                        "validated": True
                    })
            
            elif result['pattern_name'] == 'IN_PAN':
                for match in result['matches']:
                    findings.append({
                        "pattern_name": "IN_PAN", 
                        "matches": [match],
                        "sample_text": result['sample_text'],
                        "confidence_score": result['validation_method']['confidence'],
                        "validation_method": json.dumps(result['validation_method']),
                        "validated": True
                    })
            
            elif result['pattern_name'] == 'EMAIL_ADDRESS':
                for match in result['matches']:
                    findings.append({
                        "pattern_name": "EMAIL_ADDRESS",
                        "matches": [match], 
                        "sample_text": result['sample_text'],
                        "confidence_score": result['validation_method']['confidence'],
                        "validation_method": json.dumps(result['validation_method']),
                        "validated": True
                    })
    
    # Create scan payload
    payload = {
        "scan_id": f"real_scan_{int(time.time())}",
        "scan_metadata": {
            "scanner_version": "hawk-eye-scanner-2.0-real",
            "scan_timestamp": datetime.now().isoformat(),
            "intelligence_at_edge": True,
            "sdk_validated": True,
            "data_source": "real_filesystem",
            "file_path": "/Users/prathameshyadav/ARC-Hawk/apps/scanner/real_test_data/customer_data.txt"
        },
        "findings": findings
    }
    
    print(f"🚀 Ingesting {len(findings)} real PII findings to backend...")
    print(f"📊 Scan ID: {payload['scan_id']}")
    
    # Send to backend
    try:
        response = requests.post(
            ingest_url,
            json=payload,
            headers={'Content-Type': 'application/json'},
            timeout=30
        )
        
        if response.status_code == 200:
            print("✅ SUCCESS: Real PII findings ingested successfully!")
            print(f"📈 Response: {response.json()}")
        else:
            print(f"❌ FAILED: {response.status_code} - {response.text}")
            
    except Exception as e:
        print(f"❌ ERROR: Failed to connect to backend: {e}")

if __name__ == "__main__":
    ingest_real_scan_results()