"""
Credit Card Recognizer with Luhn Validation
============================================
Detects credit card numbers with mathematical validation.
"""

import re
from typing import Optional, List
from presidio_analyzer import Pattern, PatternRecognizer
import sys
import os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from validators import Luhn, is_dummy_data


class CreditCardRecognizer(PatternRecognizer):
    """Custom recognizer for credit cards with Luhn validation."""
    
    PATTERNS = [
        Pattern(
            name="Credit Card (Visa/MC/Amex/Discover/JCB/Diners)",
            # Fixed: last group is \d{3,4} to capture 15-digit Amex (and Diners)
            # Added: JCB 3528-3589, Diners Club 2131/1800
            regex=(
                r"\b(?:4[0-9]{3}|5[1-5][0-9]{2}|3[47][0-9]{2}|"
                r"6(?:011|5[0-9]{2})|35[2-8][0-9]|(?:2131|1800))"
                r"[\s-]?[0-9]{4}[\s-]?[0-9]{4}[\s-]?[0-9]{3,4}\b"
            ),
            score=0.3
        ),
    ]
    
    CONTEXT = [
        "card",
        "credit card",
        "debit card",
        "payment",
        "visa",
        "mastercard",
        "amex",
    ]
    
    def __init__(self):
        super().__init__(
            supported_entity="CREDIT_CARD",
            name="Credit Card Recognizer (Luhn)",
            supported_language="en",
            patterns=self.PATTERNS,
            context=self.CONTEXT
        )
    
    def validate_result(self, pattern_text: str) -> Optional[bool]:
        """Validate with Luhn algorithm."""
        clean = re.sub(r"[^0-9]", "", pattern_text)
        
        if len(clean) < 13 or len(clean) > 19:
            return False
        
        if is_dummy_data(clean):
            return False
        
        return Luhn.validate(clean)


if __name__ == "__main__":
    recognizer = CreditCardRecognizer()
    print(f"Credit Card Recognizer created: {recognizer.name}")
