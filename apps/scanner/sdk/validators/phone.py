"""
Indian Phone Number Validator
==============================
Validates 10-digit Indian mobile phone numbers.

Format: Must start with 6, 7, 8, or 9 (mobile number prefix)
Length: Exactly 10 digits
"""

import re


class IndianPhoneValidator:
    """Validates Indian phone numbers (10 digits, mobile format)."""
    
    # Valid mobile prefixes in India (6, 7, 8, 9)
    VALID_PREFIXES = {'6', '7', '8', '9'}
    
    # Regex for basic validation
    PHONE_PATTERN = re.compile(r'^\d{10}$')
    
    @classmethod
    def validate(cls, phone: str) -> bool:
        """
        Validates an Indian phone number.
        
        Args:
            phone: Phone number string (may contain spaces, hyphens, +91)
            
        Returns:
            True if valid, False otherwise
        """
        if not phone:
            return False
        
        # Clean the number (remove spaces, hyphens, country code)
        clean = cls._clean_phone(phone)
        
        # Must be exactly 10 digits
        if not cls.PHONE_PATTERN.match(clean):
            return False
        
        # Must start with valid mobile prefix
        if clean[0] not in cls.VALID_PREFIXES:
            return False
        
        # Reject obviously invalid patterns
        if cls._is_invalid_pattern(clean):
            return False
        
        return True
    
    @staticmethod
    def _clean_phone(phone: str) -> str:
        """Remove formatting and country code from phone number."""
        # Remove common formatting
        clean = phone.replace(' ', '').replace('-', '').replace('(', '').replace(')', '')
        
        # Remove +91 country code if present
        if clean.startswith('+91'):
            clean = clean[3:]
        elif clean.startswith('91') and len(clean) == 12:
            clean = clean[2:]
        elif clean.startswith('0') and len(clean) == 11:
            # Remove leading 0 (old STD format)
            clean = clean[1:]
        
        return clean
    
    @staticmethod
    def _is_invalid_pattern(phone: str) -> bool:
        """
        Entropy Filter: Reject dummy/sequential patterns.
        
        Anti-Dummy Patterns:
        - All same digits (9999999999)
        - Sequential increasing (0123456789, 1234567890, etc.)
        - Sequential decreasing (9876543210)
        - Repeating pairs (1212121212)
        """
        # All same digits
        if len(set(phone)) == 1:
            return True
        
        # Sequential increasing (any starting point)
        for start in range(10):
            sequential = ''.join(str((start + i) % 10) for i in range(10))
            if phone == sequential:
                return True
        
        # Sequential decreasing  
        for start in range(10):
            sequential = ''.join(str((start - i) % 10) for i in range(10))
            if phone == sequential:
                return True
        
        # Repeating pairs (e.g., 1212121212)
        if len(phone) == 10:
            pair = phone[:2]
            if phone == pair * 5:
                return True
        
        return False


def validate_indian_phone(phone: str) -> bool:
    """
    Validates an Indian phone number.
    
    Convenience function wrapping IndianPhoneValidator.validate()
    
    Args:
        phone: Phone number string
        
    Returns:
        True if valid, False otherwise
    """
    return IndianPhoneValidator.validate(phone)


if __name__ == "__main__":
    print("=== Indian Phone Validator Tests ===\n")
    
    test_cases = [
        ("9876543210", True, "Valid mobile"),
        ("8765432109", True, "Valid mobile starting with 8"),
        ("7654321098", True, "Valid mobile starting with 7"),
        ("6543210987", True, "Valid mobile starting with 6"),
        ("+91 9876543210", True, "Valid with country code"),
        ("91 9876543210", True, "Valid with 91 prefix"),
        ("09876543210", True, "Valid with leading 0"),
        ("5876543210", False, "Invalid prefix (5)"),
        ("987654321", False, "Too short"),
        ("98765432109", False, "Too long"),
        ("9999999999", False, "All same digits"),
        ("0123456789", False, "Sequential pattern"),
        ("1234567890", False, "Invalid prefix"),
    ]
    
    for phone, expected, description in test_cases:
        result = validate_indian_phone(phone)
        status = "✓" if result == expected else "✗"
        print(f"{status} {description}: {phone}")
        print(f"   Expected: {expected}, Got: {result}\n")
