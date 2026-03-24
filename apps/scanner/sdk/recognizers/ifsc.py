"""
IFSC Code Recognizer
====================
Detects IFSC (Indian Financial System Code).

Format: 4 letters + 0 + 6 alphanumeric (e.g., SBIN0001234)
"""

from typing import Optional
from presidio_analyzer import Pattern, PatternRecognizer
from sdk.validators.ifsc import validate_ifsc


class IFSCRecognizer(PatternRecognizer):
    """Custom recognizer for IFSC codes."""
    
    PATTERNS = [
        Pattern(
            name="IFSC Code (AAAA0999999)",
            regex=r"(?i)\b[A-Z]{4}0[A-Z0-9]{6}\b",
            score=0.6
        ),
    ]
    
    CONTEXT = [
        "ifsc",
        "ifsc code",
        "bank code",
        "branch code",
        "swift",
        "routing",
    ]
    
    def __init__(self):
        super().__init__(
            supported_entity="IN_IFSC",
            name="IFSC Code Recognizer",
            supported_language="en",
            patterns=self.PATTERNS,
            context=self.CONTEXT
        )
    
    def validate_result(self, pattern_text: str) -> Optional[bool]:
        """Validate IFSC format using strict validator."""
        # Use the actual IFSC validator
        return validate_ifsc(pattern_text)


if __name__ == "__main__":
    recognizer = IFSCRecognizer()
    print(f"IFSC Recognizer created: {recognizer.name}")
