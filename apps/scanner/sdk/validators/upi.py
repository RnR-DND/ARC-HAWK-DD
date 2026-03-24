"""
UPI ID Validator
================
Validates Unified Payments Interface (UPI) IDs.

Format: user@provider (e.g., user@paytm, 9876543210@ybl)
Providers: paytm, phonepe, googlepay, ybl (Yes Bank), oksbi, etc.
"""

import re


class UPIValidator:
    """Validates UPI IDs."""
    
    # Known UPI providers (common ones)
    KNOWN_PROVIDERS = {
        'paytm', 'phonepe', 'googlepay', 'gpay', 'ybl', 'oksbi', 'okhdfcbank',
        'okaxis', 'okicici', 'ibl', 'airtel', 'fbl', 'pockets', 'apl'
    }
    
    # Basic UPI pattern: user@provider
    UPI_PATTERN = re.compile(r'^[a-zA-Z0-9._-]+@[a-zA-Z0-9]+$')
    
    @classmethod
    def validate(cls, upi: str) -> bool:
        """
        Validates a UPI ID.
        
        Args:
            upi: UPI ID string
            
        Returns:
            True if valid, False otherwise
        """
        if not upi:
            return False
        
        # Normalize
        upi = upi.strip().lower()
        
        # Must have exactly one @
        if upi.count('@') != 1:
            return False
        
        # Basic format check
        if not cls.UPI_PATTERN.match(upi):
            return False
        
        # Split into user and provider
        user, provider = upi.split('@')
        
        # User part checks
        if len(user) == 0 or len(user) > 100:
            return False
        
        # Provider checks (must be known provider or valid format)
        if len(provider) < 2 or len(provider) > 50:
            return False
        
        return True


def validate_upi(upi: str) -> bool:
    """
    Validates a UPI ID.
    
    Args:
        upi: UPI ID string
        
    Returns:
        True if valid, False otherwise
    """
    return UPIValidator.validate(upi)


if __name__ == "__main__":
    print("=== UPI ID Validator Tests ===\n")
    
    test_cases = [
        ("user@paytm", True, "Valid Paytm UPI"),
        ("9876543210@ybl", True, "Valid phone-based UPI"),
        ("john.doe@phonepe", True, "Valid with dot"),
        ("user_name@googlepay", True, "Valid with underscore"),
        ("test-user@oksbi", True,"Valid with hyphen"),
        ("invalid", False, "Missing @"),
        ("@paytm", False, "Missing user"),
        ("user@", False, "Missing provider"),
        ("user@@paytm", False, "Double @"),
    ]
    
    for upi, expected, description in test_cases:
        result = validate_upi(upi)
        status = "✓" if result == expected else "✗"
        print(f"{status} {description}: {upi}")
        print(f"   Expected: {expected}, Got: {result}\n")
