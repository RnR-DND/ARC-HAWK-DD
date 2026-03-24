from hawk_scanner.internals.scanner_engine import ContextAwareScanner
import json

# Define patterns
patterns = {
    "AWS_ACCESS_KEY_ID": r"(A3T[A-Z0-9]|AKIA|AGPA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{16}",
    "GENERIC_KEY": r"['\"]([a-zA-Z0-9!@#$%^&*()_+]{20,})['\"]"
}

# Test content covering different scenarios
content = """
# Scenario 1: High Confidence Secret (Variable Assignment + High Entropy + Keyword)
# Valid format AWS Key
AWS_ACCESS_KEY_ID = "AKIAIOSFODNN7EXAMPLE12"

# Scenario 2: Test Data (Should be rejected/Low Score)
# Contains 'example'
api_key = "example_key_value_12345"

# Scenario 3: Low Entropy Secret (Should score lower)
password = "password123password123"

# Scenario 4: Commented Secret (Should score lower)
# AWS_SECRET = "AKIAIOSFODNN7EXAMPLE12"
"""

print("Running Context-Aware Scan...")
scanner = ContextAwareScanner(debug=False)
findings = scanner.scan(content, patterns, "test_source")

print(json.dumps(findings, indent=2))
