import requests
import sys
import json

BASE_URL = "http://localhost:8080"
FAILURES = []

def test(name, method, url, payload=None, expected_status=200):
    try:
        full_url = f"{BASE_URL}{url}"
        print(f"[{name}] {method} {url}...", end=" ", flush=True)
        
        if method == "GET":
            response = requests.get(full_url, timeout=5)
        elif method == "POST":
            response = requests.post(full_url, json=payload, timeout=5)
            
        if response.status_code == expected_status:
            print(f"✅ OK")
            return response
        else:
            print(f"❌ FAIL (Got {response.status_code})")
            print(f"   Response: {response.text[:200]}")
            FAILURES.append(name)
            return None
    except Exception as e:
        print(f"❌ ERROR: {e}")
        FAILURES.append(name)
        return None

print("=== ARC Hawk Endpoint Verification ===\n")

# 1. Health Check
test("Health", "GET", "/health")

# 2. Classification Summary
test("Class Summary", "GET", "/api/v1/classification/summary")

# 3. Lineage (Default)
r = test("Lineage (Default)", "GET", "/api/v1/lineage")

# 4. Lineage (System Level)
test("Lineage (System)", "GET", "/api/v1/lineage?level=system")

# 5. Findings
test("Findings", "GET", "/api/v1/findings")

# 6. Asset Details (Dynamic)
if r and r.json().get('data'):
    nodes = r.json()['data'].get('nodes', [])
    asset_id = None
    for n in nodes:
        # Assuming asset types are not reserved keywords
        if n.get('type') not in ['system', 'finding', 'classification']:
            asset_id = n.get('id')
            break
            
    if asset_id:
        test("Asset Context", "GET", f"/api/v1/assets/{asset_id}")
    else:
        print("⚠️ No Asset found to test /assets/:id")
else:
    print("⚠️ Lineage failed, skipping Asset Context")

print("\n=== Summary ===")
if len(FAILURES) == 0:
    print("✅ All Tests Passed")
    sys.exit(0)
else:
    print(f"❌ {len(FAILURES)} Tests Failed: {', '.join(FAILURES)}")
    sys.exit(1)
