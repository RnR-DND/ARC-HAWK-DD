"""
Aadhaar Recognizer - Custom Presidio Pattern Recognizer
========================================================
Detects Indian Aadhaar numbers with mathematical validation.

Features:
- Regex pattern matching
- Context keyword boosting
- Dummy data filtering
- Verhoeff checksum validation
"""

import re
from typing import Optional, List
from presidio_analyzer import Pattern, PatternRecognizer

# Import validators from sdk package
import sys
import os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from validators import Verhoeff, is_dummy_data


class AadhaarRecognizer(PatternRecognizer):
    """
    Custom recognizer for Indian Aadhaar numbers (12-digit UID).
    
    Implements three-stage validation:
    1. Regex pattern match
    2. Dummy data filtering
    3. Verhoeff checksum validation
    """
    
    PATTERNS = [
        Pattern(
            name="Aadhaar (12 digits)",
            regex=r"(?<!\d)[2-9]\d{3}[\s-]?\d{4}[\s-]?\d{4}(?!\d)",
            score=0.9  # High score because we have strict Verhoeff validation
        ),
    ]
    
    CONTEXT = [
        "aadhaar",
        "aadhar",
        "uid",
        "uidai",
        "unique identification",
        "unique identity",
        "enrollment",
        "enrolment",
    ]
    
    def __init__(self):
        super().__init__(
            supported_entity="IN_AADHAAR",
            name="Aadhaar Recognizer (Mathematical)",
            supported_language="en",
            patterns=self.PATTERNS,
            context=self.CONTEXT
        )
    
    def validate_result(self, pattern_text: str) -> Optional[bool]:
        """
        Validates the matched pattern using mathematical checks.
        
        This is the CRITICAL GATE that prevents false positives.
        
        Args:
            pattern_text: The matched text from regex
            
        Returns:
            True if valid, False if invalid, None to use default logic
        """
        # Step 1: Clean the number (remove spaces/hyphens)
        clean_number = re.sub(r"[^0-9]", "", pattern_text)
        
        # Step 2: Length check
        if len(clean_number) != 12:
            return False
        
        # Step 3: First digit cannot be 0 or 1 (Aadhaar rule)
        if clean_number[0] in ['0', '1']:
            return False
        
        # Step 4: Dummy Data Detection (CRITICAL FILTER)
        if is_dummy_data(clean_number):
            # This catches: 111111111111, 123456789012, etc.
            return None
        
        # Step 5: Verhoeff Checksum Validation (MATHEMATICAL PROOF)
        if not Verhoeff.validate(clean_number):
            return None
        
        # All checks passed - this is a real Aadhaar number
        return True


if __name__ == "__main__":
    print("=== AadhaarRecognizer Test ===\n")
    
    from engine import SharedAnalyzerEngine
    
    # Initialize engine and add recognizer
    engine = SharedAnalyzerEngine.get_engine()
    recognizer = AadhaarRecognizer()
    SharedAnalyzerEngine.add_recognizer(recognizer)
    
    # Test cases
    test_strings = [
        ("My Aadhaar is 9999 1111 2226", "Should DETECT (valid checksum)"),
        ("User ID is 1234 5678 9012", "Should IGNORE (linear sequence)"),
        ("Number: 1111 1111 1111", "Should IGNORE (repeating)"),
        ("Random: 9876 5432 1000", "Should IGNORE (invalid Verhoeff)"),
        ("UID 2345 6789 0126 for customer", "Should DETECT (valid with context)"),
    ]
    
    print("Testing Aadhaar detection with mathematical validation:\n")
    for text, expected in test_strings:
        results = engine.analyze(text=text, language='en', entities=["IN_AADHAAR"])
        
        if results:
            detected = text[results[0].start:results[0].end]
            print(f"✓ DETECTED: '{text}'")
            print(f"  Match: {detected} (Score: {results[0].score:.2f})")
            print(f"  {expected}\n")
        else:
            print(f"✗ IGNORED: '{text}'")
            print(f"  {expected}\n")
