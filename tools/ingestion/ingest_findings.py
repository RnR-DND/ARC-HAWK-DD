#!/usr/bin/env python3
"""
ingest_findings.py - Findings Ingestion Tool

Ingests PII findings into backend API.
"""

import sys
import json
import requests
from typing import Dict, Any, List, Optional


class FindingsIngestor:
    """Ingests findings into the ARC-Hawk backend"""
    
    def __init__(self, api_url: str = "http://localhost:8081/api/v1"):
        self.api_url = api_url
        self.ingest_endpoint = f"{api_url}/scans/ingest-verified"
    
    def ingest(self, findings: Dict[str, List[Dict[str, Any]]]) -> Dict[str, Any]:
        """
        Ingest findings to backend.
        
        Args:
            findings: Dict with source types as keys (fs, postgresql, etc.)
        
        Returns:
            Response from backend
        """
        # Validate findings
        if not findings or all(len(v) == 0 for v in findings.values()):
            return {"error": "No findings provided"}
        
        # Validate each finding
        for source_type, finding_list in findings.items():
            for finding in finding_list:
                if not finding.get("verified"):
                    return {"error": f"Unverified finding rejected: {finding.get('pattern_name')}"}
        
        # Send to backend
        try:
            response = requests.post(
                self.ingest_endpoint,
                json=findings,
                headers={"Content-Type": "application/json"},
                timeout=60
            )
            return {
                "status_code": response.status_code,
                "response": response.json()
            }
        except requests.RequestException as e:
            return {"error": str(e)}
    
    def test_connection(self) -> bool:
        """Test backend connectivity"""
        try:
            response = requests.get(f"{self.api_url}/health", timeout=10)
            return response.status_code == 200
        except requests.RequestException:
            return False


def execute(findings: Dict[str, List[Dict[str, Any]]], api_url: Optional[str] = None) -> Dict[str, Any]:
    """Execute findings ingestion"""
    ingestor = FindingsIngestor(api_url or "http://localhost:8081/api/v1")
    return ingestor.ingest(findings)


def main():
    """CLI entry point"""
    if len(sys.argv) < 2:
        print("Usage: ingest_findings.py <findings.json>")
        sys.exit(1)
    
    with open(sys.argv[1], 'r') as f:
        findings = json.load(f)
    
    result = execute(findings)
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
