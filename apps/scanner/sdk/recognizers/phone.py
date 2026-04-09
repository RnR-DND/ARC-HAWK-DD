"""
Indian Phone Number Recognizer
===============================
Detects Indian mobile and landline phone numbers.

Formats:
  +91-XXXXXXXXXX  International prefix (high confidence)
  0XX-XXXXXXXX    STD landline with area code (medium confidence)
  XXXXXXXXXX      10-digit mobile, 6-9 prefix (low base confidence;
                  requires phone-related column name for reliable detection)

IMPORTANT — Broad Pattern Notice:
The bare 10-digit mobile pattern matches any 10-digit number starting with 6-9.
Without phone-related column context, confidence is lowered to 0.2 to avoid
flooding results with false positives from numeric IDs, order numbers, etc.
"""

from typing import Optional
from presidio_analyzer import Pattern, PatternRecognizer
from sdk.validators.phone import validate_indian_phone


class IndianPhoneRecognizer(PatternRecognizer):
    """Custom recognizer for Indian mobile and landline phone numbers."""

    PATTERNS = [
        Pattern(
            name="Indian Phone (+91 international prefix)",
            # +91 or 0091 followed by 10-digit mobile number
            regex=r"(?i)(?:\+91|0{2}91)[\s\-]?[6-9][0-9]{9}\b",
            score=0.85   # High: unambiguous country code prefix
        ),
        Pattern(
            name="Indian Phone (STD landline 0XX-XXXXXXXX)",
            # STD code: 2-4 digits starting with 0, followed by 6-8 digit subscriber
            regex=r"\b0[1-9][0-9]{1,3}[\s\-]?[0-9]{6,8}\b",
            score=0.6    # Medium: STD format is specific but still somewhat broad
        ),
        Pattern(
            name="Indian Phone (10-digit mobile, context-gated)",
            # Mobile: starts with 6/7/8/9, exactly 10 digits
            # Low base score — should only surface with phone column context
            regex=r"\b[6-9][0-9]{9}\b",
            score=0.2    # Very low base; context boost brings to 0.7
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
        "contact_number",
        "phone_no",
        "mobile_no",
        "tel",
        "landline",
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
