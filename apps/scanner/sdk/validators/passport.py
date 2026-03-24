"""
Indian Passport Number Validator
=================================
Validates Indian passport numbers.

Format: Starts with uppercase letter, followed by 7 digits
Example: A1234567, Z9876543
"""

import re


class IndianPassportValidator:
    """Validates Indian passport numbers with type checking."""
    
    # Passport types (first character)
    DIPLOMATIC_TYPES = {'J', 'Z'}  # Diplomatic/Official
    PERSONAL_TYPES = set('ABCDEFGHIKLMNOPQRSTUVWX')  # Regular Personal (A-W, excluding J, Z)
    
    # Format: Letter + 7 digits
    PASSPORT_PATTERN = re.compile(r'^[A-Z][0-9]{7}$')
    
    @classmethod
    def validate(cls, passport: str) -> bool:
        """
        Validates an Indian passport number.
        
        Alpha-Numeric Traffic Logic:
        - First Character: Passport type
          J, Z = Diplomatic/Official contexts
          A-W = Regular Personal Passports
        - Following 7 characters: Strictly numeric
        
        Args:
            passport: Passport number string
            
        Returns:
            True if valid, False otherwise
        """
        if not passport:
            return False
        
        # Normalize (uppercase, remove spaces)
        clean = passport.upper().replace(' ', '').replace('-', '')
        
        # Check pattern
        if not cls.PASSPORT_PATTERN.match(clean):
            return False
        
        # Validate first character (passport type)
        first_char = clean[0]
        if first_char not in (cls.DIPLOMATIC_TYPES | cls.PERSONAL_TYPES):
            return False
        
        return True


def validate_indian_passport(passport: str) -> bool:
    """
    Validates an Indian passport number.
    
    Args:
        passport: Passport number string
        
    Returns:
        True if valid, False otherwise
    """
    return IndianPassportValidator.validate(passport)


if __name__ == "__main__":
    print("=== Indian Passport Validator Tests ===\n")
    
    test_cases = [
        ("A1234567", True, "Valid passport"),
        ("Z9876543", True, "Valid passport with Z"),
        ("M5432109", True, "Valid passport with M"),
        ("a1234567", True, "Valid (lowercase converted)"),
        ("A 1234567", True, "Valid with space"),
        ("12345678", False, "Missing letter"),
        ("AB123456", False, "Two letters"),
        ("A12345", False, "Too few digits"),
        ("A123456789", False, "Too many digits"),
    ]
    
    for passport, expected, description in test_cases:
        result = validate_indian_passport(passport)
        status = "✓" if result == expected else "✗"
        print(f"{status} {description}: {passport}")
        print(f"   Expected: {expected}, Got: {result}\n")
