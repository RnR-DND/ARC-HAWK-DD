"""
PAN Recognizer - Permanent Account Number (India)
==================================================
Detects Indian PAN numbers with format validation.

Format: AAAAA9999A (5 letters, 4 digits, 1 letter)
4th character indicates type: P=Person, C=Company, H=HUF, F=Firm, etc.
"""

import re
from typing import Optional, List
from presidio_analyzer import Pattern, PatternRecognizer


class PANRecognizer(PatternRecognizer):
    """Custom recognizer for Indian PAN numbers with strict validation."""
    
    PATTERNS = [
        Pattern(
            name="PAN (AAAAA9999A)",
            regex=r"(?i)\b[A-Z]{5}[0-9]{4}[A-Z]\b",
            score=0.9  # High score because we have strict format+checksum validation
        ),
    ]
    
    CONTEXT = [
        "pan",
        "pancard",
        "pan card",
        "permanent account",
        "income tax",
        "tax id",
        "tax number",
    ]
    
    # Valid 4th character types (entity type indicator)
    VALID_ENTITY_TYPES = {
        'P',  # Individual/Person
        'C',  # Company
        'H',  # Hindu Undivided Family (HUF)
        'F',  # Firm/Partnership
        'A',  # Association of Persons (AOP)
        'T',  # Trust
        'B',  # Body of Individuals (BOI)
        'L',  # Local Authority
        'J',  # Artificial Juridical Person
        'G',  # Government
    }
    
    def __init__(self):
        super().__init__(
            supported_entity="IN_PAN",
            name="PAN Recognizer",
            supported_language="en",
            patterns=self.PATTERNS,
            context=self.CONTEXT
        )
    
    def validate_result(self, pattern_text: str) -> Optional[bool]:
        """
        Strict PAN validation using the enhanced validator with anti-fake checks.
        
        This now delegates to sdk.validators.pan.PANValidator which includes:
        - Check digit validation (Weighted Modulo 26)
        - Entity type validation
        - Anti-fake pattern rejection (AAAAA, ABCDE, etc.)
        - Test data context rejection
        """
        from sdk.validators.pan import PANValidator
        
        # Note: Context will be empty here since Presidio doesn't pass it
        # The validator will still catch obvious fakes (AAAAA, ABCDE, wrong check digits)
        # Context-based filtering happens post-detection in the ingestion layer
        return PANValidator.validate(pattern_text, context="")


if __name__ == "__main__":
    recognizer = PANRecognizer()
    print(f"PAN Recognizer created: {recognizer.name}")
    
    # Test validation
    test_cases = [
        ("ABCDE1234F", True, "Valid format with P"),
        ("AAAPC1234D", True, "Valid company PAN"),
        ("AAAPZ1234Q", False, "Invalid entity type Z"),
        ("ABCD123456", False, "Too many digits"),
    ]
    
    for pan, expected, desc in test_cases:
        result = recognizer.validate_result(pan)
        status = "✓" if result == expected else "✗"
        print(f"{status} {desc}: {pan} -> {result}")
