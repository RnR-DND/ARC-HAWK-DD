#!/usr/bin/env python3
"""
map_dpdpa.py - DPDPA 2023 Compliance Mapper

Maps PII types to DPDPA categories and checks compliance.
"""

import sys
import json
import requests
from typing import Dict, Any, List, Optional


class DPDPAMapper:
    """Maps PII types to DPDPA 2023 categories"""
    
    DPDPA_MAPPING = {
        "IN_AADHAAR": {
            "category": "Sensitive Personal Data",
            "requires_consent": True,
            "retention_period_days": 1825  # 5 years
        },
        "IN_PAN": {
            "category": "Financial Identifier",
            "requires_consent": True,
            "retention_period_days": 2555  # 7 years
        },
        "CREDIT_CARD": {
            "category": "Financial Identifier",
            "requires_consent": True,
            "retention_period_days": 730  # 2 years
        },
        "IN_PASSPORT": {
            "category": "Sensitive Personal Data",
            "requires_consent": True,
            "retention_period_days": 3650  # 10 years
        },
        "EMAIL": {
            "category": "Contact Information",
            "requires_consent": True,
            "retention_period_days": 365  # 1 year
        },
        "PHONE": {
            "category": "Contact Information",
            "requires_consent": True,
            "retention_period_days": 365  # 1 year
        },
        "BANK_ACCOUNT": {
            "category": "Financial Identifier",
            "requires_consent": True,
            "retention_period_days": 2555  # 7 years
        },
        "IFSC": {
            "category": "Financial Identifier",
            "requires_consent": True,
            "retention_period_days": 2555  # 7 years
        }
    }
    
    def __init__(self, api_url: str = "http://localhost:8081/api/v1"):
        self.api_url = api_url
    
    def map_pii_type(self, pii_type: str) -> Dict[str, Any]:
        """Map a PII type to DPDPA category"""
        return self.DPDPA_MAPPING.get(pii_type, {
            "category": "Other",
            "requires_consent": False,
            "retention_period_days": 365
        })
    
    def map_findings(self, findings: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """Map DPDPA categories to findings"""
        for finding in findings:
            pii_type = finding.get("pattern_name", "")
            mapping = self.map_pii_type(pii_type)
            finding["dpdpa_category"] = mapping["category"]
            finding["requires_consent"] = mapping["requires_consent"]
            finding["retention_period_days"] = mapping["retention_period_days"]
        return findings
    
    def get_compliance_overview(self) -> Dict[str, Any]:
        """Get compliance overview from backend"""
        try:
            response = requests.get(
                f"{self.api_url}/compliance/overview",
                timeout=10
            )
            return response.json()
        except requests.RequestException as e:
            return {"error": str(e)}
    
    def get_consent_violations(self) -> List[Dict[str, Any]]:
        """Get consent violations"""
        try:
            response = requests.get(
                f"{self.api_url}/compliance/violations",
                timeout=10
            )
            return response.json().get("data", [])
        except requests.RequestException as e:
            return [{"error": str(e)}]
    
    def get_retention_violations(self) -> List[Dict[str, Any]]:
        """Get retention violations"""
        try:
            response = requests.get(
                f"{self.api_url}/retention/violations",
                timeout=10
            )
            return response.json().get("data", [])
        except requests.RequestException as e:
            return [{"error": str(e)}]


def execute(findings: Optional[List[Dict[str, Any]]] = None, api_url: Optional[str] = None) -> Dict[str, Any]:
    """Execute DPDPA mapping"""
    mapper = DPDPAMapper(api_url or "http://localhost:8081/api/v1")
    
    if findings:
        mapped = mapper.map_findings(findings)
        return {"mapped_findings": mapped}
    else:
        return mapper.get_compliance_overview()


def main():
    """CLI entry point"""
    if len(sys.argv) > 1:
        with open(sys.argv[1], 'r') as f:
            findings = json.load(f)
        result = execute(findings)
    else:
        result = execute()
    
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
