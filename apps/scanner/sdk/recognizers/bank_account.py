"""
Bank Account Number Recognizer
===============================
Detects Indian bank account numbers.

Format: 9-18 digits

IMPORTANT — False Positive Risk: "critical"
The bare 9-18 digit pattern matches almost any numeric field.
Base confidence is 0.1; only boosted to 0.7 when the column/table
name contains a known bank account context keyword.

High-confidence column name patterns:
  account_number, acc_no, bank_account, acct_num, a/c, account_no
"""

from typing import Optional
from presidio_analyzer import Pattern, PatternRecognizer
from sdk.validators.bank_account import validate_bank_account


# Column/table name substrings that strongly suggest a bank account field.
# When Presidio matches and surrounding text contains one of these, the
# context_enhancer in AnalyzerEngine will boost the score.
HIGH_CONFIDENCE_CONTEXT = [
    "account_number",
    "acc_no",
    "bank_account",
    "acct_num",
    "a/c",
    "account_no",
    "bank_acc",
    "acctno",
]


class BankAccountRecognizer(PatternRecognizer):
    """
    Custom recognizer for Indian bank account numbers.

    Uses an extremely low base confidence (0.1) because the 9-18 digit
    pattern is dangerously broad.  The score is only meaningful when
    context keywords (column names like account_number, acc_no, etc.)
    are present — Presidio's context mechanism boosts the score in those cases.
    """

    PATTERNS = [
        Pattern(
            name="Bank Account (9-18 digits, context-gated)",
            regex=r"\b[0-9]{9,18}\b",
            # Very low base score — essentially requires context boost to surface
            score=0.1
        ),
    ]

    CONTEXT = HIGH_CONFIDENCE_CONTEXT + [
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
