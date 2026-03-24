"""
Bank Account Number Recognizer
===============================
Detects Indian bank account numbers.

Format: 9-18 digits
"""

from typing import Optional
from presidio_analyzer import Pattern, PatternRecognizer
from sdk.validators.bank_account import validate_bank_account


class BankAccountRecognizer(PatternRecognizer):
    """Custom recognizer for Indian bank account numbers."""
    
    PATTERNS = [
        Pattern(
            name="Bank Account (9-18 digits)",
            regex=r"\b\d{9,18}\b",
            score=0.3  # Low score as many numbers match this
        ),
    ]
    
    CONTEXT = [
        "account",
        "bank account",
        "account number",
        "acc no",
        "account no",
        "savings account",
        "current account",
        "banking",
    ]
    
    def __init__(self):
        super().__init__(
            supported_entity="IN_BANK_ACCOUNT",
            name="Bank Account Recognizer",
            supported_language="en",
            patterns=self.PATTERNS,
            context=self.CONTEXT
        )
    
    def validate_result(self, pattern_text: str) -> Optional[bool]:
        """Validate bank account format using strict validator."""
        # Use the actual bank account validator
        return validate_bank_account(pattern_text)


if __name__ == "__main__":
    recognizer = BankAccountRecognizer()
    print(f"Bank Account Recognizer created: {recognizer.name}")
