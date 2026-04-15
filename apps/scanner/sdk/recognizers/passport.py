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
            name="Passport (A1234567)",
            regex=r"(?i)(?:passport\s*(?:no|number)?[:\s]*)?\b[A-Z][0-9]{7}\b",
            score=0.6
        ),
    ]
    
    CONTEXT = [
        "passport",
        "travel",
        "document",
        "passport number",
        "passport no",
        "travel document",
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
