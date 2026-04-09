"""
Indian Passport Recognizer
===========================
Detects Indian passport numbers.

Format: Letter + 7 digits (e.g., A1234567)
"""

from typing import Optional
from presidio_analyzer import Pattern, PatternRecognizer
from sdk.validators.passport import validate_indian_passport


class IndianPassportRecognizer(PatternRecognizer):
    """Custom recognizer for Indian passport numbers."""
    
    PATTERNS = [
        Pattern(
            name="Indian Passport (improved format)",
            # [A-PR-WY]: excludes Q, X, Z (not issued as first letter by MEA)
            # [1-9]: second char is non-zero digit
            # [0-9]{5}: five digits
            # [1-9]: seventh char non-zero digit
            # [A-Za-z]: final check letter
            regex=r"(?i)\b[A-PR-WYa-pr-wy][1-9][0-9]{5}[1-9][A-Za-z]\b",
            score=0.75   # Improved from 0.5: more specific format
        ),
    ]

    CONTEXT = [
        "passport",
        "travel",
        "document",
        "passport number",
        "passport no",
        "travel document",
        "passport_no",
        "pp_no",
        "passport_number",
    ]
    
    def __init__(self):
        super().__init__(
            supported_entity="IN_PASSPORT",
            name="Indian Passport Recognizer",
            supported_language="en",
            patterns=self.PATTERNS,
            context=self.CONTEXT
        )
    
    def validate_result(self, pattern_text: str) -> Optional[bool]:
        """Validate passport format using strict validator."""
        # Use the actual passport validator
        return validate_indian_passport(pattern_text)


if __name__ == "__main__":
    recognizer = IndianPassportRecognizer()
    print(f"Passport Recognizer created: {recognizer.name}")
