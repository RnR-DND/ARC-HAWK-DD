#!/usr/bin/env python3
"""
build_graph.py - Lineage Graph Builder

Builds Neo4j graph from findings data.
"""

import sys
import json
import requests
from typing import Dict, Any, List, Optional


class LineageGraphBuilder:
    """Builds and maintains lineage graph in Neo4j"""
    
    def __init__(self, api_url: str = "http://localhost:8081/api/v1"):
        self.api_url = api_url
        self.sync_endpoint = f"{api_url}/lineage/sync"
    
    def sync_from_findings(self, findings: Dict[str, List[Dict[str, Any]]]) -> Dict[str, Any]:
        """
        Sync lineage graph from findings.
        
        Args:
            findings: Dict with source types as keys
        
        Returns:
            Sync result
        """
        try:
            response = requests.post(
                self.sync_endpoint,
                json={"findings": findings},
                headers={"Content-Type": "application/json"},
                timeout=60
            )
            return {
                "status_code": response.status_code,
                "response": response.json()
            }
        except requests.RequestException as e:
            return {"error": str(e)}
    
    def get_graph_stats(self) -> Dict[str, Any]:
        """Get lineage graph statistics"""
        try:
            response = requests.get(
                f"{self.api_url}/lineage/stats",
                timeout=10
            )
            return response.json()
        except requests.RequestException as e:
            return {"error": str(e)}
    
    def get_semantic_graph(self) -> Dict[str, Any]:
        """Get full semantic graph"""
        try:
            response = requests.get(
                f"{self.api_url}/graph/semantic",
                timeout=30
            )
            return response.json()
        except requests.RequestException as e:
            return {"error": str(e)}


def execute(findings: Dict[str, List[Dict[str, Any]]], api_url: Optional[str] = None) -> Dict[str, Any]:
    """Execute lineage sync"""
    builder = LineageGraphBuilder(api_url or "http://localhost:8081/api/v1")
    return builder.sync_from_findings(findings)


def main():
    """CLI entry point"""
    if len(sys.argv) < 2:
        print("Usage: build_graph.py <findings.json>")
        sys.exit(1)
    
    with open(sys.argv[1], 'r') as f:
        findings = json.load(f)
    
    result = execute(findings)
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
