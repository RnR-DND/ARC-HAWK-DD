"""
Bank Account Number Validator
==============================
Validates Indian bank account numbers.

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
    def validate(cls, account: str) -> bool:
        """
        Validates a bank account number.
        
        Args:
            account: Account number string
            
        Returns:
            True if valid, False otherwise
        """
        if not account:
            return False
        
        # Clean the number (remove spaces, hyphens)
        clean = account.replace(' ', '').replace('-', '')
        
        # Must be digits only
        if not cls.ACCOUNT_PATTERN.match(clean):
            return False
        
        # Check length
        if len(clean) < cls.MIN_LENGTH or len(clean) > cls.MAX_LENGTH:
            return False
        
        # Reject obviously invalid patterns
        if cls._is_invalid_pattern(clean):
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


def validate_bank_account(account: str) -> bool:
    """
    Validates a bank account number.
    
    Args:
        account: Account number string
        
    Returns:
        True if valid, False otherwise
    """
    return BankAccountValidator.validate(account)


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
