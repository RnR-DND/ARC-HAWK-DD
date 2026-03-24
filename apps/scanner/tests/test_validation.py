import pytest
from hawk_scanner.internals.validation_integration import validate_findings

def test_validate_findings():
    matches = [{'pattern_name': 'EMAIL', 'matches': ['john@company.com'], 'sample_text': 'john@company.com'}]
    args = type('Args', (), {'verbose': False, 'quiet': False, 'debug': False})()
    validated = validate_findings(matches, args)
    assert len(validated) == 1
    assert validated[0]['pattern_name'] == 'EMAIL'