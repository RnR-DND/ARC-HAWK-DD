"""
Locked PII Scope - Intelligence-at-Edge Architecture
====================================================
Only these 11 India PII types are allowed for processing.
All other PII types must be rejected by the scanner.

Language: English only
Compliance: India DPDPA 2023
"""

from typing import Set


# ==================================================================================
# LOCKED PII SCOPE - Non-negotiable
# ==================================================================================
# These 11 PII types are the ONLY ones the system is certified to handle.
# Adding new types requires legal/compliance review.
# ==================================================================================

LOCKED_PII_TYPES: Set[str] = {
    "IN_PAN",              # Permanent Account Number (India Tax ID)
    "IN_PASSPORT",         # Indian Passport Number
    "IN_AADHAAR",          # Aadhaar (Unique Identification Number)
    "CREDIT_CARD",         # Credit/Debit Card Numbers
    "IN_UPI",              # Unified Payments Interface ID
    "IN_IFSC",             # Indian Financial System Code
    "IN_BANK_ACCOUNT",     # Bank Account Number
    "IN_PHONE",            # Indian Phone Number (10 digits)
    "EMAIL_ADDRESS",       # Email Address
    "IN_VOTER_ID",         # Voter ID (EPIC - Electors Photo Identity Card)
    "IN_DRIVING_LICENSE",  # Indian Driving License
}


def is_allowed_pii(pii_type: str) -> bool:
    """
    Check if a PII type is in the locked scope.
    
    Args:
        pii_type: PII type identifier (e.g., "IN_AADHAAR", "CREDIT_CARD")
        
    Returns:
        True if PII type is allowed, False otherwise
    """
    normalized = pii_type.upper().strip()
    return normalized in LOCKED_PII_TYPES


def get_locked_types() -> Set[str]:
    """
    Get the complete set of locked PII types.
    
    Returns:
        Set of all allowed PII type identifiers
    """
    return LOCKED_PII_TYPES.copy()


def validate_pii_type_or_raise(pii_type: str) -> None:
    """
    Validate PII type is in locked scope, raise exception if not.
    
    Args:
        pii_type: PII type to validate
        
    Raises:
        ValueError: If PII type is not in locked scope
    """
    if not is_allowed_pii(pii_type):
        raise ValueError(
            f"PII type '{pii_type}' is not in locked scope. "
            f"Only these types are allowed: {', '.join(sorted(LOCKED_PII_TYPES))}"
        )


if __name__ == "__main__":
    print("=== Locked PII Scope ===\n")
    print(f"Total locked PII types: {len(LOCKED_PII_TYPES)}\n")
    
    print("Allowed PII types:")
    for pii_type in sorted(LOCKED_PII_TYPES):
        print(f"  ✓ {pii_type}")
    
    print("\n=== Validation Tests ===\n")
    
    # Test valid types
    test_valid = ["IN_AADHAAR", "CREDIT_CARD", "EMAIL_ADDRESS"]
    for pii_type in test_valid:
        result = is_allowed_pii(pii_type)
        print(f"✓ {pii_type}: {result}")
    
    # Test invalid types
    print()
    test_invalid = ["US_SSN", "UK_NHS", "GENERIC_SECRET"]
    for pii_type in test_invalid:
        result = is_allowed_pii(pii_type)
        print(f"✗ {pii_type}: {result} (correctly rejected)")
