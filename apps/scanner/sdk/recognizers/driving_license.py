"""
Indian Driving License Recognizer
==================================
Detects Indian driving license numbers.

Format: 2 letters (state) + 13 digits (e.g., MH0120150001234)
"""

from typing import Optional
from presidio_analyzer import Pattern, PatternRecognizer
from sdk.validators.driving_license import validate_driving_license


class DrivingLicenseRecognizer(PatternRecognizer):
    """Custom recognizer for Indian driving licenses."""
    
    PATTERNS = [
        Pattern(
            name="Driving License (AA9999999999999)",
            regex=r"(?i)\b[A-Z]{2}[-\s]?[0-9]{13}\b",
            score=0.5
        ),
    ]
    
    CONTEXT = [
        "driving license",
        "driving licence",
        "dl",
        "license",
        "licence",
        "dl number",
        "driver",
    ]
    
    def __init__(self):
        super().__init__(
            supported_entity="IN_DRIVING_LICENSE",
            name="Driving License Recognizer",
            supported_language="en",
            patterns=self.PATTERNS,
            context=self.CONTEXT
        )
    
    def validate_result(self, pattern_text: str) -> Optional[bool]:
        """Validate Driving License format using strict validator."""
        # Use the actual driving license validator
        return validate_driving_license(pattern_text)


if __name__ == "__main__":
    recognizer = DrivingLicenseRecognizer()
    print(f"Driving License Recognizer created: {recognizer.name}")
