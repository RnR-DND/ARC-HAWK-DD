"""
Indian Phone Number Recognizer
===============================
Detects Indian mobile phone numbers.

Format: 10 digits starting with 6, 7, 8, or 9
"""

from typing import Optional
from presidio_analyzer import Pattern, PatternRecognizer
from sdk.validators.phone import validate_indian_phone


class IndianPhoneRecognizer(PatternRecognizer):
    """Custom recognizer for Indian phone numbers."""
    
    PATTERNS = [
        Pattern(
            name="Indian Phone (flexible)",
            regex=r"(?<!\d)(?:\+91[- ]?|0|91)?([6-9]\d{9})(?!\d)",
            score=0.8
        ),
    ]
    
    CONTEXT = [
        "phone",
        "mobile",
        "contact",
        "cell",
        "telephone",
        "number",
        "phone number",
        "mobile number",
    ]
    
    def __init__(self):
        super().__init__(
            supported_entity="IN_PHONE",
            name="Indian Phone Recognizer",
            supported_language="en",
            patterns=self.PATTERNS,
            context=self.CONTEXT
        )
    
    def validate_result(self, pattern_text: str) -> Optional[bool]:
        """Validate phone format using strict validator."""
        # Use the actual phone validator
        return validate_indian_phone(pattern_text)


if __name__ == "__main__":
    recognizer = IndianPhoneRecognizer()
    print(f"Phone Recognizer created: {recognizer.name}")
