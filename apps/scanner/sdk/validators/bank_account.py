""" Bank Account Number Validator ============================== Validates 
Indian bank account numbers.

Format: Variable length, typically 9-18 digits
No universal checksum algorithm (bank-specific)
"""

import re


class BankAccountValidator:
    """Validates Indian bank account numbers."""

    # Most Indian bank accounts are 9-18 digits
    MIN_LENGTH = 9
    MAX_LENGTH = 18

    # Pattern: digits only
    ACCOUNT_PATTERN = re.compile(r'^\d+$')

    @classmethod
    def validate(cls, account: str, context: str = "") -> bool:

        if not account:
            return False

        clean = re.sub(r'\D', '', account)

        # 🚨 MUST be digits
        if not clean.isdigit():
            return False

        # 🚨 length check
        if not (9 <= len(clean) <= 18):
            return False

        # 🚨 reject all same digits
        if len(set(clean)) == 1:
            return False

        # 🚨 reject phone like patterns
        if len(clean) == 10 and clean[0] in "6789":
            return False

        if len(clean) == 12:
            return False

        return True

    @staticmethod
    def _is_invalid_pattern(account: str) -> bool:
        """Check for obviously invalid patterns."""
        # All same digits
        if len(set(account)) == 1:
            return True

        # All zeros
        if account == '0' * len(account):
            return True

        return False


def validate_bank_account(account: str, context: str = "") -> bool:
    """
    Validates a bank account number.

    Args:
        account: Account number string

    Returns:
        True if valid, False otherwise
    """
    return BankAccountValidator.validate(account, context)


if __name__ == "__main__":
    print("=== Bank Account Validator Tests ===\n")

    test_cases = [
        ("123456789012", True, "Valid 12-digit account"),
        ("987654321098765", True, "Valid 15-digit account"),
        ("1234567890", True, "Valid 10-digit account"),
        ("12345678", False, "Too short (8 digits)"),
        ("1234567890123456789", False, "Too long (19 digits)"),
        ("000000000000", False, "All zeros"),
        ("111111111111", False, "All same digit"),
        ("12AB34567890", False, "Contains letters"),
        ("1234 5678 9012", True, "Valid with spaces"),
    ]

    for account, expected, description in test_cases:
        result = validate_bank_account(account)
        status = "✓" if result == expected else "✗"
        print(f"{status} {description}: {account}")
        print(f"   Expected: {expected}, Got: {result}\n")
 
