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
            regex=r"[A-Za-z]{4}[\s-]?0[\s-]?[A-Za-z0-9]{6}",
            score=1.0
        ),
    ]
    
    CONTEXT = []
    
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
        clean = pattern_text.replace(" ","").replace("-","").upper()
        return validate_ifsc(clean)


if __name__ == "__main__":
    recognizer = IFSCRecognizer()
    print(f"IFSC Recognizer created: {recognizer.name}")
