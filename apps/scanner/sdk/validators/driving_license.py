"""
Indian Driving License Validator
=================================
Validates Indian driving license numbers.

Format varies by state, but generally:
- State code (2 letters) + District code (2 digits) + Year (4 digits) + Serial (7 digits)
- Example: MH0120150001234, DL0720180005678

Alternative format: State code + 13 digits
"""

import re


class DrivingLicenseValidator:
    """Validates Indian driving license numbers."""
    
    # Common format: 2 letters + 13 digits (state + issue year + serial)
    DL_PATTERN_COMMON = re.compile(r'^[A-Z]{2}[0-9]{13}$')
    
    # Alternative format: 2 letters + hyphen/space + 13 digits
    DL_PATTERN_ALT = re.compile(r'^[A-Z]{2}[-\s]?[0-9]{13}$')
    
    # Valid Indian state codes (sample - not exhaustive)
    VALID_STATE_CODES = {
        'AN', 'AP', 'AR', 'AS', 'BR', 'CH', 'CG', 'DD', 'DL', 'GA',
        'GJ', 'HP', 'HR', 'JH', 'JK', 'KA', 'KL', 'LA', 'LD', 'MH',
        'ML', 'MN', 'MP', 'MZ', 'NL', 'OD', 'OR', 'PB', 'PY', 'RJ',
        'SK', 'TN', 'TR', 'TS', 'UK', 'UP', 'WB',
    }
    
    @classmethod
    def validate(cls, dl: str) -> bool:
        """
        Validates an Indian driving license number.
        
        Args:
            dl: Driving license string
            
        Returns:
            True if valid, False otherwise
        """
        if not dl:
            return False
        
        # Normalize (uppercase, preserve hyphens/spaces initially)
        clean = dl.upper().strip()
        
        # Check alternative format first
        if cls.DL_PATTERN_ALT.match(clean):
            # Remove hyphen/space for further checks
            clean = clean.replace('-', '').replace(' ', '')
        
        # Must be 15 characters (2 letters + 13 digits)
        if len(clean) != 15:
            return False
        
        # Check pattern
        if not cls.DL_PATTERN_COMMON.match(clean):
            return False
        
        # Validate state code (optional - can be disabled for flexibility)
        # state_code = clean[:2]
        # if state_code not in cls.VALID_STATE_CODES:
        #     return False
        
        return True


def validate_driving_license(dl: str) -> bool:
    """
    Validates an Indian driving license number.
    
    Args:
        dl: Driving license string
        
    Returns:
        True if valid, False otherwise
    """
    return DrivingLicenseValidator.validate(dl)


if __name__ == "__main__":
    print("=== Driving License Validator Tests ===\n")
    
    test_cases = [
        ("MH0120150001234", True, "Valid Maharashtra DL"),
        ("DL0720180005678", True, "Valid Delhi DL"),
        ("KA1220190009876", True, "Valid Karnataka DL"),
        ("mh0120150001234", True, "Valid (lowercase converted)"),
        ("MH-0120150001234", True, "Valid with hyphen"),
        ("MH 0120150001234", True, "Valid with space"),
        ("M0120150001234", False, "Only 1 letter"),
        ("MH012015000123", False, "Too few digits"),
        ("MH01201500012345", False, "Too many digits"),
        ("1H0120150001234", False, "Starts with digit"),
    ]
    
    for dl, expected, description in test_cases:
        result = validate_driving_license(dl)
        status = "✓" if result == expected else "✗"
        print(f"{status} {description}: {dl}")
        print(f"   Expected: {expected}, Got: {result}\n")
