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
from validators.luhn import validate_credit_card
from validators import is_dummy_data

class CreditCardRecognizer(PatternRecognizer):
    """Custom recognizer for credit cards with Luhn validation."""
    
    PATTERNS = [
        Pattern(
            name="Credit Card (Visa/MC/Amex/Discover)",
            regex=r"\b(?:\d[ -]*?){13,19}\b",
            score=0.85
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
        
        return validate_credit_card(clean)


if __name__ == "__main__":
    recognizer = CreditCardRecognizer()
    print(f"Credit Card Recognizer created: {recognizer.name}")
