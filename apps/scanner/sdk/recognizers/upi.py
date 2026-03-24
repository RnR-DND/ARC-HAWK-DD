"""
UPI ID Recognizer
=================
Detects UPI (Unified Payments Interface) IDs.

Format: user@provider (e.g., user@paytm, 9876543210@ybl)
"""

from typing import Optional
from presidio_analyzer import Pattern, PatternRecognizer
from sdk.validators.upi import validate_upi


class UPIRecognizer(PatternRecognizer):
    """Custom recognizer for UPI IDs."""
    
    PATTERNS = [
        Pattern(
            name="UPI ID (user@provider)",
            regex=r"(?i)\b[a-z0-9._-]+@(?:paytm|phonepe|googlepay|gpay|ybl|oksbi|okaxis|okicici|okhdfcbank|ibl|airtel|apl|fbl)\b",
            score=0.6
        ),
        Pattern(
            name="UPI ID (generic)",
            regex=r"(?i)\b[a-z0-9._-]+@[a-z0-9]+\b",
            score=0.3
        ),
    ]
    
    CONTEXT = [
        "upi",
        "upi id",
        "payment",
        "paytm",
        "phonepe",
        "gpay",
        "google pay",
        "transfer",
    ]
    
    def __init__(self):
        super().__init__(
            supported_entity="IN_UPI",
            name="UPI ID Recognizer",
            supported_language="en",
            patterns=self.PATTERNS,
            context=self.CONTEXT
        )
    
    def validate_result(self, pattern_text: str) -> Optional[bool]:
        """Validate UPI format using strict validator."""
        # Use the actual UPI validator
        return validate_upi(pattern_text)


if __name__ == "__main__":
    recognizer = UPIRecognizer()
    print(f"UPI Recognizer created: {recognizer.name}")
