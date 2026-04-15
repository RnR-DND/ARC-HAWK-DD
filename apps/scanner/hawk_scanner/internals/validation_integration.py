"""
Scanner Validation Integration
===============================
Integrates SDK validators into the main scanner pipeline.

This module bridges the gap between the regex-based pattern matching
and the mathematical validators in the SDK.

INTELLIGENCE-AT-EDGE: Only validated findings are returned.
"""

from sdk.validators import IndianPassportValidator
from sdk.validators import validate_email, IndianPhoneValidator
from sdk.validators import validate_aadhaar, validate_credit_card, validate_pan
from sdk.validators.upi import validate_upi
from sdk.validators.bank_account import validate_bank_account
from sdk.validators.ifsc import validate_ifsc
from sdk.engine import SharedAnalyzerEngine
from sdk.validators.driving_license import validate_driving_license
from sdk.validators.voter_id import validate_voter_id
import re
import sys
import inspect
import argparse
from typing import Optional, List, Dict, Any
from pathlib import Path
from presidio_analyzer import PatternRecognizer, Pattern

sys.path.insert(0, str(Path(__file__).parent.parent.parent))

VALIDATOR_MAP = {
    "IN_AADHAAR": validate_aadhaar,
    "IN_PAN": validate_pan,
    "CREDIT_CARD": validate_credit_card,
    "EMAIL_ADDRESS": validate_email,
    "IN_PHONE": IndianPhoneValidator.validate,
    "IN_PASSPORT": IndianPassportValidator.validate,
    "IN_BANK_ACCOUNT": validate_bank_account,
    "IFSC": validate_ifsc,
    "IFSC_CODE": validate_ifsc,
    "IN_IFSC": validate_ifsc,
    "IN_DRIVING_LICENSE": validate_driving_license,
    "IN_VOTER_ID": validate_voter_id,
    "IN_UPI": validate_upi
}

PII_TYPE_PATTERNS = {
    'AADHAAR': [r'(?:^|[^0-9])([2-9][0-9]{11})(?![0-9])'],
    'PAN': [r'(?:^|[^A-Z])([A-Z]{5}[0-9]{4}[A-Z])(?![A-Z0-9])'],
    'CREDIT_CARD': [
        r'\b(?:\d[ -]*?){13,19}\b',
        r'\b(?:4\d{12,18}|5[1-5]\d{14}|3[47]\d{13})\b'
    ],
    'EMAIL': [r'(?:^|[^A-Za-z0-9])([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})(?![a-zA-Z0-9._%+-])'],
    'PHONE': [r'(?<!\d)(\+91[\s-]?[6-9]\d{9}|[6-9]\d{9})(?!\d)'],
    'UPI': [r'(?:^|[^a-zA-Z0-9])([a-zA-Z0-9._-]+@[a-zA-Z0-9._-]+)(?![a-zA-Z0-9._-])'],
    'IFSC': [r'(?:^|[^A-Za-z0-9])([A-Z]{4}[\s-]?0[\s-]?[A-Z0-9]{6})(?![A-Za-z0-9])'],
    'IN_BANK_ACCOUNT': [r'(?:^|[^0-9])([0-9]{9,18})(?![0-9])'],
    'PASSPORT': [r'(?:^|[^A-Za-z0-9])([A-Za-z][0-9]{7})(?![A-Za-z0-9])'],
    'VOTER_ID': [r'(?:^|[^A-Z0-9])([A-Z]{3}[0-9]{7})(?![A-Z0-9])'],
    'DRIVING_LICENSE': [r'([A-Z]{2}[-\s]?\d{1,2}[-\s]?\d{4}[-\s]?\d{7})']
}

# Strict PII Scope - Only these types are allowed
ALLOWED_PII_TYPES = {
    'IN_AADHAAR',
    'IN_PAN',
    'CREDIT_CARD',
    'EMAIL_ADDRESS',
    'IN_PHONE',
    'IN_PASSPORT',
    'IN_BANK_ACCOUNT',
    'IFSC',
    'IFSC_CODE',
    'IN_IFSC',
    'IN_DRIVING_LICENSE',
    'IN_VOTER_ID',
    'IN_UPI'
}

# Mapping from fingerprint.yml pattern names (uppercase) to Validator Keys
PATTERN_MAPPING = {
    'AADHAR': 'IN_AADHAAR',
    'AADHAAR': 'IN_AADHAAR',
    'PAN': 'IN_PAN',
    'PASSPORT_INDIA': 'IN_PASSPORT',
    'PHONE_INDIA': 'IN_PHONE',
    'EMAIL': 'EMAIL_ADDRESS',
    'CREDIT_CARD_VISA': 'CREDIT_CARD',
    'CREDIT_CARD_MC': 'CREDIT_CARD',
    'CREDIT_CARD_AMEX': 'CREDIT_CARD',
    'CREDIT_CARD_DISCOVER': 'CREDIT_CARD'
}


def get_validator_for_pattern(pattern_name: str):
    """Get the validator function for a pattern name."""
    # Fix: Validator map keys are uppercase
    pattern_upper = pattern_name.upper()

    # Normalize pattern name using mapping if exists
    normalized_name = PATTERN_MAPPING.get(pattern_upper, pattern_upper)

    return VALIDATOR_MAP.get(normalized_name)

def get_normalized_name(pattern_name: str) -> str:
    """Get normalized pattern name for scope checking"""
    pattern_upper = pattern_name.upper()
    return PATTERN_MAPPING.get(pattern_upper, pattern_upper)

def validate_match(value: str, pattern_name: str, full_text: str = "") -> tuple[bool, str]:
    """
    Validate a match using the appropriate validator.

    Args:
        value: The matched value to validate
        pattern_name: The pattern name (e.g., 'Aadhaar', 'PAN')

    Returns:
        Tuple of (is_valid, validation_method)
    """
    print("VALIDATOR DEBUG:", pattern_name, get_validator_for_pattern(pattern_name))
    # 1. SCOPE CHECK: Enforce strict PII locking
    normalized_name = get_normalized_name(pattern_name)

    if normalized_name not in ALLOWED_PII_TYPES:
        # Check if original was allowed (just in case)
        if pattern_name.upper() not in ALLOWED_PII_TYPES:
            return False, 'scope_rejected'

    validator = get_validator_for_pattern(pattern_name)

    # 2. VALIDATOR CHECK: Fail closed if no validator exists
    if validator is None:
        # In strict mode, we should reject.
        # But if it's in ALLOWED_PII_TYPES but missing validator map entry, that's a bug or config issue.
        # Given we have validators for all ALLOWED types above, this is safe.
        return False, 'no_validator'

    try:
        params = inspect.signature(validator).parameters
        if len(params) ==1:
            is_valid = validator(value)
        else:
            is_valid = validator(value, full_text)
        method = validator.__name__
        return is_valid, method
    except Exception as e:
        # 3. EXCEPTION HANDLING: Fail closed
        print(f"[VALIDATION ERROR] {pattern_name}: {e}")
        return False, 'error'

def validate_findings(findings: List[Dict[str, Any]], args=None, strict_mode: bool = False) -> List[Dict[str, Any]]:
    """
    Validate findings using SDK validators.

    This implements INTELLIGENCE-AT-EDGE by:
    1. Checking if a validator exists for the pattern
    2. Running mathematical/format validation
    3. Filtering out invalid findings

    Args:
        findings: List of finding dictionaries from match_strings()
        args: Command line arguments for verbose output
        strict_mode: If True, reject findings without validators

    Returns:
        List of validated findings only
    """
    validated_findings = []
    total_original = sum(len(f.get('matches', [])) for f in findings)

    normalized_seen = set()
    for finding in findings:
        pattern_name = finding.get('pattern_name', '')
        matches = finding.get('matches', [])
        validated_matches = []
        validation_info = {}

        for match in matches:
            if pattern_name == "IN_PASSPORT":
                # 🔥 Passport-specific normalization (SAFE)
                clean = match.upper()
                m = re.search(r'[A-Z][0-9]{7}', clean)
                clean = m.group() if m else match.strip()

            else:
                clean = match.strip()

            unique_key = (pattern_name, clean)

            # Deduplicate globally
            if unique_key in normalized_seen:
                continue

            is_valid, method = validate_match(clean, pattern_name, str(finding))

            if is_valid:
                normalized_seen.add(unique_key)
                validated_matches.append(match)

                if method not in validation_info:
                    validation_info[method] = []

                validation_info[method].append(match)

        if validated_matches:
            finding_copy = finding.copy()
            finding_copy['matches'] = validated_matches
            finding_copy['validation_method'] = validation_info
            finding_copy['original_match_count'] = len(matches)
            finding_copy['validated_match_count'] = len(validated_matches)
            validated_findings.append(finding_copy)

        sum(len(f.get('matches', [])) for f in validated_findings)

    return validated_findings


def validate_and_enhance_result(result: Dict[str, Any], args=None) -> Optional[Dict[str, Any]]:
    """
    Validate a single result and enhance with validation info.

    Args:
        result: Single finding result
        args: Command line arguments

    Returns:
        Enhanced result or None if invalid
    """
    pattern_name = result.get('pattern_name', '')
    matches = result.get('matches', [])

    if not matches:
        return result

    validated_matches = []
    for match in matches:
        is_valid, method = validate_match(match, pattern_name)
        if is_valid:
            validated_matches.append(clean)

    if not validated_matches:
        if args and hasattr(args, 'debug') and args.debug:
            print(f"[VALIDATION] All matches rejected for {pattern_name}")
        return None

    result['matches'] = validated_matches
    result['validation_method'] = method
    return result


def run_validated_scan(args, content: str, source: str = 'text') -> List[Dict[str, Any]]:
    """
    Run a complete validated scan with SDK validation.

    This is a replacement for system.match_strings() that includes
    intelligence-at-edge validation.

    Args:
        args: Command line arguments
        content: Text content to scan
        source: Source identifier

    Returns:
        List of validated findings
    """
    engine = SharedAnalyzerEngine.get_engine()
    results = engine.analyze(
        text=content,
        language="en",
        entities=[
            "IFSC",
            "IN_AADHAAR",
            "IN_PAN",
            "IN_PHONE",
            "EMAIL_ADDRESS",
            "CREDIT_CARD",
            "IN_BANK_ACCOUNT",
            "IN_PASSPORT"
        ]
    )
    findings = []
    print("RAW PRESIDIO RESULTS:", results)

    for r in results:
        match_text = content[r.start:r.end]

        print("ENTITY:", r.entity_type, "VALUE:", match_text)

        findings.append({
            "data_source": source,
            "pattern_name": r.entity_type,
            "matches": [match_text],
            "score": r.score,
        })

    print("DEBUG ENTITY:", r.entity_type, "TEXT:", content[r.start:r.end])

    for f in findings:
        print("DETECTED TYPE:", f.get("pattern_name"))
    print("DEBUG PRESIDIO FINDINGS:", findings)
    validated_results = validate_findings(findings, args)
    return validated_results

if __name__ == '__main__':

    parser = argparse.ArgumentParser(
        description='Test scanner validation integration')
    parser.add_argument('--test-value', help='Value to test')
    parser.add_argument('--test-pattern', help='Pattern name to test')
    parser.add_argument('--strict', action='store_true',
                        help='Strict validation mode')
    args = parser.parse_args(
        ['--test-value', '999911112226', '--test-pattern', 'aadhaar'])

    if args.test_value and args.test_pattern:
        is_valid, method = validate_match(args.test_value, args.test_pattern)
        print(f"Test: {args.test_value} against {args.test_pattern}")
        print(f"Valid: {is_valid}, Method: {method}")
    else:
        print("Testing all validators...")

        test_cases = [
            ('999911112226', 'aadhaar'),
            ('ABCDE1234F', 'pan'),
            ('4532015112830366', 'credit_card'),
            ('test@example.com', 'email'),
            ('+919876543210', 'phone'),
            ('abc@upi', 'upi'),
            ('HDFC0001234', 'ifsc'),
            ('123456789012', 'bank_account'),
        ]

        for value, pattern in test_cases:
            is_valid, method = validate_match(value, pattern)
            print(
                f"{pattern}: {value[:15]}... -> Valid: {is_valid}, Method: {method}")
