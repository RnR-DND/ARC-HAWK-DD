"""
Validation Pipeline - Intelligence-at-Edge (Enhanced)
=====================================================
Maps PII types to validators and creates VerifiedFinding objects.

This is the core integration point that ensures ONLY validated PIIs
reach the backend. Invalid findings are filtered out here.

ENHANCEMENTS:
- Context-aware validation to reduce false positives
- Test data detection and filtering
- Confidence adjustment based on surrounding keywords
"""

from typing import Optional, Callable
from sdk.schema import VerifiedFinding, SourceInfo

# Import all validators
from sdk.validators.luhn import validate_credit_card
from sdk.validators.verhoeff import validate_aadhaar
from sdk.validators.pan import validate_pan
from sdk.validators.phone import validate_indian_phone
from sdk.validators.email import validate_email
from sdk.validators.passport import validate_indian_passport
from sdk.validators.upi import validate_upi
from sdk.validators.ifsc import validate_ifsc
from sdk.validators.bank_account import validate_bank_account
from sdk.validators.voter_id import validate_voter_id
from sdk.validators.driving_license import validate_driving_license
from sdk.validators.international import validate_gst_checksum

# Import context validator for enhanced detection
from sdk.validators.context_validator import ContextValidator

# Import PII scope checker
from sdk.pii_scope import is_allowed_pii

# Initialize global context validator
_context_validator = ContextValidator()


# Validator mapping: PII type -> validator function
VALIDATOR_MAP = {
    "CREDIT_CARD": validate_credit_card,
    "IN_AADHAAR": validate_aadhaar,
    "IN_PAN": validate_pan, 
    "IN_PHONE": validate_indian_phone,
    "EMAIL_ADDRESS": validate_email,
    "IN_PASSPORT": validate_indian_passport,
    "IN_UPI": validate_upi,
    "IN_IFSC": validate_ifsc,
    "IN_BANK_ACCOUNT": validate_bank_account,
    "IN_VOTER_ID": validate_voter_id,
    "IN_DRIVING_LICENSE": validate_driving_license,
}


def get_validator(pii_type: str) -> Optional[Callable[[str], bool]]:
    """
    Get validator function for a PII type.
    
    Args:
        pii_type: PII type identifier (e.g., "IN_AADHAAR", "CREDIT_CARD")
        
    Returns:
        Validator function or None if not found
    """
    return VALIDATOR_MAP.get(pii_type.upper())


def validate_and_create_finding(presidio_result, text: str, source_info: SourceInfo, pattern_name: str) -> Optional[VerifiedFinding]:
    """
    Validate a Presidio result and create VerifiedFinding if valid.
    
    Enhanced Intelligence-at-Edge Workflow:
    1. Check if PII type is in locked scope
    2. Get validator for PII type
    3. Run mathematical/format validation
    4. Run context-aware validation (NEW: test data detection, confidence adjustment)
    5. If valid, create VerifiedFinding with adjusted confidence
    6. If invalid, return None (finding is REJECTED)
    
    Args:
        presidio_result: Presidio RecognizerResult
        text: Full text that was analyzed
        source_info: Source location information
        pattern_name: Original pattern name
        
    Returns:
        VerifiedFinding if valid, None if invalid/rejected
    """
    pii_type = presidio_result.entity_type
    
    # Step 1: Check PII scope
    if not is_allowed_pii(pii_type):
        print(f"⚠️  Rejected {pii_type}: Not in locked scope (11 India PIIs only)")
        return None
    
    # Extract matched value
    matched_value = text[presidio_result.start:presidio_result.end]
    
    # Step 2: Get validator
    validator = get_validator(pii_type)
    
    if validator:
        # Step 3: Run mathematical validation
        is_valid = validator(matched_value)
        
        if not is_valid:
            print(f"⚠️  Rejected {pii_type}: Failed {validator.__name__} validation")
            return None
        
        validators_passed = [validator.__name__]
        validation_method = "mathematical"  # Luhn, Verhoeff, etc.
        base_confidence = presidio_result.score
    else:
        # No validator available (shouldn't happen for locked PIIs)
        print(f"⚠️  Warning: No validator for {pii_type}")
        validators_passed = []
        validation_method = "ml"
        base_confidence = presidio_result.score
    
    # Step 4: Context-aware validation (NEW)
    is_context_valid, adjusted_confidence, rejection_reason = _context_validator.validate_with_context(
        value=matched_value,
        pii_type=pii_type,
        text=text,
        start=presidio_result.start,
        end=presidio_result.end,
        base_confidence=base_confidence
    )
    
    if not is_context_valid:
        print(f"⚠️  {rejection_reason}")
        return None
    
    # Step 5: Create VerifiedFinding with adjusted confidence
    verified = VerifiedFinding.create_from_analysis(
        presidio_result=presidio_result,
        text=text,
        source_info=source_info,
        pattern_name=pattern_name,
        validators=validators_passed
    )

    # Update confidence score with context-adjusted value
    verified.confidence_score = adjusted_confidence

    # Phase 3: Context window (±50 chars) for human review
    ctx_start = max(0, presidio_result.start - 50)
    ctx_end = min(len(text), presidio_result.end + 50)
    verified.context_window = text[ctx_start:ctx_end]

    # Phase 3: matched_by array — which layers validated this finding
    verified.matched_by = validators_passed if validators_passed else ["ml"]
    if adjusted_confidence != base_confidence:
        verified.matched_by.append("context_adjuster")

    # Phase 3: Unicode normalization tag (NFC)
    import unicodedata
    original = matched_value
    normalized = unicodedata.normalize("NFC", matched_value)
    verified.unicode_normalized = original != normalized

    confidence_change = adjusted_confidence - base_confidence
    confidence_indicator = "↑" if confidence_change > 0 else "↓" if confidence_change < 0 else "="

    print(f"✅ Verified {pii_type}: {matched_value[:10]}*** "
          f"(confidence: {adjusted_confidence:.2f} {confidence_indicator}, "
          f"validators: {len(validators_passed)})")

    return verified


def filter_and_validate_results(presidio_results, text: str, source_info: SourceInfo, pattern_name: str) -> list[VerifiedFinding]:
    """
    Filter Presidio results through validation pipeline.
    
    Args:
        presidio_results: List of Presidio RecognizerResults
        text: Full text
        source_info: Source information
        pattern_name: Pattern name
        
    Returns:
        List of VerifiedFinding objects (only valid findings)
    """
    verified_findings = []
    
    for result in presidio_results:
        verified = validate_and_create_finding(result, text, source_info, pattern_name)
        if verified:
            verified_findings.append(verified)
    
    total = len(presidio_results)
    valid = len(verified_findings)
    rejected = total - valid
    
    print(f"📊 Validation Results: {valid}/{total} valid ({rejected} rejected)")
    
    return verified_findings


def set_exclusion_list(exclusion_values: list[str]) -> None:
    """
    Set exclusion list for context validator.
    
    Args:
        exclusion_values: List of values to exclude from detection
    """
    global _context_validator
    _context_validator.add_to_exclusion_list(exclusion_values)
    print(f"📋 Added {len(exclusion_values)} values to exclusion list")


def get_validation_statistics() -> dict:
    """
    Get validation statistics from context validator.
    
    Returns:
        Dictionary with validation statistics
    """
    return _context_validator.get_statistics()


if __name__ == "__main__":
    print("=== Validation Pipeline Test ===\n")
    
    # Test validator mapping
    print("Available validators:")
    for pii_type, validator in VALIDATOR_MAP.items():
        print(f"  {pii_type}: {validator.__name__}")
    
    # Test getting validator
    print("\nTest get_validator:")
    aadhaar_validator = get_validator("IN_AADHAAR")
    print(f"IN_AADHAAR validator: {aadhaar_validator.__name__}")
    
    # Test validation
    print("\nTest validation:")
    test_aadhaar = "999911112226"  # Example (may not be valid Verhoeff)
    is_valid = aadhaar_validator(test_aadhaar)
    print(f"Aadhaar {test_aadhaar}: Valid = {is_valid}")
    
    # Test context validator statistics
    print("\nContext Validator Statistics:")
    stats = get_validation_statistics()
    for key, value in stats.items():
        print(f"  {key}: {value}")
