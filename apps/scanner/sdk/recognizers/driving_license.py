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
    
    def __init__(self):
        patterns = [
            Pattern(
                name="DL flexible",
                regex=r"\b[A-Z]{2}[- ]?\d{1,2}[- ]?\d{4}[- ]?\d{5,8}\b",
                score=0.9
            ),
            Pattern(
                name="DL compact",
                regex=r"\b[A-Z]{2}\d{13,15}\b",
                score=0.85
            )
        ]

        super().__init__(
            supported_entity="IN_DRIVING_LICENSE",
            name="Driving License Recognizer",
            supported_language="en",
            patterns=patterns
        )
    
    def validate_result(self, pattern_text: str) -> Optional[bool]:
        """Validate Driving License format using strict validator."""
        return validate_driving_license(pattern_text)


if __name__ == "__main__":
    recognizer = DrivingLicenseRecognizer()
    print(f"Driving License Recognizer created: {recognizer.name}")
