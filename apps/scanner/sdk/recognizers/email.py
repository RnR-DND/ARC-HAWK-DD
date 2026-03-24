"""
Email Address Recognizer
=========================
Detects email addresses.

Format: local@domain.tld
"""

from typing import Optional
from presidio_analyzer import Pattern, PatternRecognizer
from sdk.validators.email import validate_email


class EmailRecognizer(PatternRecognizer):
    """Custom recognizer for email addresses."""
    
    PATTERNS = [
        Pattern(
            name="Email Address",
            regex=r"\b[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b",
            score=0.5
        ),
    ]
    
    CONTEXT = [
        "email",
        "e-mail",
        "mail",
        "contact",
        "address",
        "inbox",
        "@"
    ]
    
    def __init__(self):
        super().__init__(
            supported_entity="EMAIL_ADDRESS",
            name="Email Address Recognizer",
            supported_language="en",
            patterns=self.PATTERNS,
            context=self.CONTEXT
        )
    
    def validate_result(self, pattern_text: str) -> Optional[bool]:
        """Validate email format using strict validator."""
        # Use the actual email validator
        return validate_email(pattern_text)


if __name__ == "__main__":
    recognizer = EmailRecognizer()
    print(f"Email Recognizer created: {recognizer.name}")
