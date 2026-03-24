#!/usr/bin/env python3
"""
ARC-Hawk Platform Smoke Tests
Comprehensive smoke testing for all components
"""

import requests
import sys
import json
from datetime import datetime

BASE_URL = "http://localhost:8080"
FRONTEND_URL = "http://localhost:3000"

def print_test(name, passed, details=""):
    """Print test result"""
    status = "✅ PASS" if passed else "❌ FAIL"
    print(f"  {status} - {name}")
    if details:
        print(f"       {details}")
    return passed

def test_health_check():
    """Test health endpoint"""
    try:
        response = requests.get(f"{BASE_URL}/health", timeout=5)
        return print_test(
            "Health Check",
            response.status_code == 200,
            f"Status: {response.status_code}"
        )
    except Exception as e:
        return print_test("Health Check", False, str(e))

def test_classification_summary():
    """Test classification summary endpoint"""
    try:
        response = requests.get(f"{BASE_URL}/api/v1/classification/summary", timeout=5)
        if response.status_code == 200:
            data = response.json()
            total = data.get('total_findings', 0)
            return print_test(
                "Classification Summary",
                True,
                f"Total findings: {total}"
            )
        return print_test("Classification Summary", False, f"Status: {response.status_code}")
    except Exception as e:
        return print_test("Classification Summary", False, str(e))

def test_lineage_graph():
    """Test lineage graph endpoint"""
    try:
        response = requests.get(f"{BASE_URL}/api/v1/lineage", timeout=5)
        if response.status_code == 200:
            data = response.json()
            nodes = len(data.get('nodes', []))
            edges = len(data.get('edges', []))
            return print_test(
                "Lineage Graph (PostgreSQL)",
                True,
                f"Nodes: {nodes}, Edges: {edges}"
            )
        return print_test("Lineage Graph (PostgreSQL)", False, f"Status: {response.status_code}")
    except Exception as e:
        return print_test("Lineage Graph (PostgreSQL)", False, str(e))

def test_semantic_graph():
    """Test semantic graph endpoint (Neo4j)"""
    try:
        response = requests.get(f"{BASE_URL}/api/v1/graph/semantic", timeout=5)
        if response.status_code == 200:
            data = response.json()
            nodes = len(data.get('nodes', []))
            edges = len(data.get('edges', []))
            return print_test(
                "Semantic Graph (Neo4j)",
                True,
                f"Nodes: {nodes}, Edges: {edges}"
            )
        return print_test("Semantic Graph (Neo4j)", False, f"Status: {response.status_code}")
    except Exception as e:
        return print_test("Semantic Graph (Neo4j)", False, str(e))

def test_findings():
    """Test findings endpoint"""
    try:
        response = requests.get(f"{BASE_URL}/api/v1/findings?limit=10", timeout=5)
        if response.status_code == 200:
            data = response.json()
            count = len(data)
            return print_test(
                "Findings Endpoint",
                True,
                f"Retrieved {count} findings"
            )
        return print_test("Findings Endpoint", False, f"Status: {response.status_code}")
    except Exception as e:
        return print_test("Findings Endpoint", False, str(e))

def test_assets():
    """Test assets endpoint"""
    try:
        response = requests.get(f"{BASE_URL}/api/v1/assets", timeout=5)
        if response.status_code == 200:
            data = response.json()
            count = data.get('total', 0)
            return print_test(
                "Assets Endpoint",
                True,
                f"Total assets: {count}"
            )
        return print_test("Assets Endpoint", False, f"Status: {response.status_code}")
    except Exception as e:
        return print_test("Assets Endpoint", False, str(e))

def test_frontend():
    """Test frontend accessibility"""
    try:
        response = requests.get(FRONTEND_URL, timeout=10)
        return print_test(
            "Frontend Accessibility",
            response.status_code == 200,
            f"Status: {response.status_code}"
        )
    except Exception as e:
        return print_test("Frontend Accessibility", False, str(e))

def test_cors():
    """Test CORS headers"""
    try:
        headers = {'Origin': 'http://localhost:3000'}
        response = requests.options(f"{BASE_URL}/api/v1/lineage", headers=headers, timeout=5)
        cors_header = response.headers.get('Access-Control-Allow-Origin', '')
        return print_test(
            "CORS Configuration",
            cors_header in ['http://localhost:3000', '*'],
            f"CORS Origin: {cors_header}"
        )
    except Exception as e:
        return print_test("CORS Configuration", False, str(e))

def main():
    print("\n" + "="*60)
    print("🧪 ARC-Hawk Platform - Comprehensive Smoke Tests")
    print("="*60)
    print(f"⏰ Started at: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")
    
    results = []
    
    print("━━━ Backend API Tests ━━━")
    results.append(test_health_check())
    results.append(test_classification_summary())
    results.append(test_lineage_graph())
    results.append(test_semantic_graph())
    results.append(test_findings())
    results.append(test_assets())
    
    print("\n━━━ Frontend Tests ━━━")
    results.append(test_frontend())
    
    print("\n━━━ Integration Tests ━━━")
    results.append(test_cors())
    
    # Summary
    print("\n" + "="*60)
    passed = sum(results)
    total = len(results)
    success_rate = (passed / total * 100) if total > 0 else 0
    
    print(f"📊 Test Summary: {passed}/{total} tests passed ({success_rate:.1f}%)")
    
    if passed == total:
        print("✅ All smoke tests PASSED!")
        print("="*60 + "\n")
        return 0
    else:
        print(f"❌ {total - passed} test(s) FAILED!")
        print("="*60 + "\n")
        return 1

if __name__ == "__main__":
    sys.exit(main())
