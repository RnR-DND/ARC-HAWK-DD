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
import re
import sys
from typing import Optional, List, Dict, Any
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent.parent))


VALIDATOR_MAP = {
    "IN_AADHAAR": validate_aadhaar,
    "IN_PAN": validate_pan,
    "CREDIT_CARD": validate_credit_card,
    "EMAIL_ADDRESS": validate_email,
    "IN_PHONE": IndianPhoneValidator.validate,
    "IN_PASSPORT": IndianPassportValidator.validate,
}


PII_TYPE_PATTERNS = {
    'AADHAAR': r'(?:^|[^0-9])([2-9]{1}[0-9]{3}[0-9]{4}[0-9]{4})(?![0-9])',
    'PAN': r'(?:^|[^A-Z])([A-Z]{5}[0-9]{4}[A-Z])(?![A-Z0-9])',
    'CREDIT_CARD': r'(?:^|[^0-9])([0-9]{4}[-\s]?[0-9]{4}[-\s]?[0-9]{4}[-\s]?[0-9]{4})(?![0-9])',
    'EMAIL': r'(?:^|[^A-Za-z0-9])([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})(?![a-zA-Z0-9._%+-])',
    'PHONE': r'(?:^|[^0-9])(\+?[91][-\s]?[6-9][0-9]{9})(?![0-9])',
    'UPI': r'(?:^|[^a-zA-Z0-9])([a-zA-Z0-9._-]+@[a-zA-Z0-9._-]+)(?![a-zA-Z0-9._-])',
    'IFSC': r'(?:^|[^A-Z0-9])([A-Z]{4}0[A-Z0-9]{6})(?![A-Z0-9])',
    'BANK_ACCOUNT': r'(?:^|[^0-9])([0-9]{9,18})(?![0-9])',
    'PASSPORT': r'(?:^|[^A-Z0-9])([A-Z]{1}[0-9]{7})(?![A-Z0-9])',
    'VOTER_ID': r'(?:^|[^A-Z0-9])([A-Z]{3}[0-9]{7})(?![A-Z0-9])',
    'DRIVING_LICENSE': r'(?:^|[^A-Z0-9])([A-Z]{2}[-\s]?[0-9]{2}[-\s]?[0-9]{4,7})(?![A-Z0-9])',
}


# Strict PII Scope - Only these types are allowed
ALLOWED_PII_TYPES = {
    'IN_AADHAAR',
    'IN_PAN',
    'CREDIT_CARD',
    'EMAIL_ADDRESS',
    'IN_PHONE',
    'IN_PASSPORT'
}

# Mapping from fingerprint.yml pattern names (uppercase) to Validator Keys
PATTERN_MAPPING = {
    'AADHAR': 'IN_AADHAAR',  # Common misspelling handled
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


def validate_match(value: str, pattern_name: str) -> tuple[bool, str]:
    """
    Validate a match using the appropriate validator.

    Args:
        value: The matched value to validate
        pattern_name: The pattern name (e.g., 'Aadhaar', 'PAN')

    Returns:
        Tuple of (is_valid, validation_method)
    """
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
        is_valid = validator(value)
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
    total_original = len(findings)

    for finding in findings:
        pattern_name = finding.get('pattern_name', '')
        matches = finding.get('matches', [])
        validated_matches = []
        validation_info = {}

        for match in matches:
            is_valid, method = validate_match(match, pattern_name)

            if is_valid:
                validated_matches.append(match)
                if method not in validation_info:
                    validation_info[method] = []
                validation_info[method].append(
                    match[:10] + '...' if len(match) > 10 else match)
            else:
                if args and hasattr(args, 'debug') and args.debug:
                    print(
                        f"[VALIDATION REJECTED] {pattern_name}: {match[:20]}...")

        if validated_matches:
            finding_copy = finding.copy()
            finding_copy['matches'] = validated_matches
            finding_copy['validation_method'] = validation_info
            finding_copy['original_match_count'] = len(matches)
            finding_copy['validated_match_count'] = len(validated_matches)
            validated_findings.append(finding_copy)

    rejected_count = total_original - len(validated_findings)

    if args and not args.quiet:
        print(
            f"[VALIDATION] {len(validated_findings)}/{total_original} findings passed validation")
        if rejected_count > 0:
            print(
                f"[VALIDATION] {rejected_count} findings rejected by SDK validators")

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
            validated_matches.append(match)

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
    from hawk_scanner.internals import system

    patterns = system.get_fingerprint_file(args)
    matched_strings = []

    for pattern_name, pattern_regex in patterns.items():
        compiled_regex = re.compile(pattern_regex, re.IGNORECASE)
        matches = re.findall(compiled_regex, content)

        if matches:
            found = {
                'data_source': source,
                'pattern_name': pattern_name,
                'matches': list(set(matches)),
                'sample_text': content[:100],
            }
            matched_strings.append(found)

    validated_results = validate_findings(matched_strings, args)

    return validated_results


if __name__ == '__main__':
    import argparse

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
