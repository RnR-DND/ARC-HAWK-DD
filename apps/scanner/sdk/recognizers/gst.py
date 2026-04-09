"""
GST Number Recognizer
=====================
Detects Indian GST (Goods and Services Tax) Identification Numbers (GSTIN).

Format: 15 characters
  - 2 digits: state code (01-37)
  - 10 chars: PAN of the taxpayer
  - 1 digit: entity number (1-9)
  - 1 char: check letter (Z by default, can be alpha/digit)
  - 1 char: checksum digit
Example: 27AAPFU0939F1ZV
"""

from typing import Optional
from presidio_analyzer import Pattern, PatternRecognizer


class GSTRecognizer(PatternRecognizer):
    """Custom recognizer for Indian GSTIN numbers."""

    # GSTIN: 2-digit state + 10-char PAN + 1-digit entity + 1-char check + 1-char checksum
    PATTERNS = [
        Pattern(
            name="GSTIN (15-char state+PAN+entity)",
            regex=r"\b[0-3][0-9][A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z]\b",
            score=0.85
        ),
    ]

    CONTEXT = [
        "gst",
        "gstin",
        "gst number",
        "gst no",
        "tax identification",
        "gst id",
        "registration number",
        "gst registration",
    ]

    def __init__(self):
        super().__init__(
            supported_entity="IN_GST",
            name="GSTIN Recognizer",
            supported_language="en",
            patterns=self.PATTERNS,
            context=self.CONTEXT
        )

    def validate_result(self, pattern_text: str) -> Optional[bool]:
        """Validate GSTIN format."""
        v = pattern_text.strip().upper()
        if len(v) != 15:
            return False
        # State code 01-37
        try:
            state = int(v[:2])
            if state < 1 or state > 37:
                return False
        except ValueError:
            return False
        # Chars 3-12: must be valid PAN format (AAAAA9999A)
        pan = v[2:12]
        import re
        if not re.match(r'^[A-Z]{5}[0-9]{4}[A-Z]$', pan):
            return False
        # Entity number: 1-9 or A-Z
        if not (v[12].isdigit() or v[12].isalpha()):
            return False
        # Position 14 (index 13): typically 'Z'
        # Position 15 (index 14): checksum - alphanumeric
        return True


if __name__ == "__main__":
    recognizer = GSTRecognizer()
    print(f"GST Recognizer created: {recognizer.name}")
