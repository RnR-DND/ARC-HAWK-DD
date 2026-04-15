"""
IFSC Code Validator
===================
Validates Indian Financial System Code (IFSC).

Format: 4 letters (bank code) + 0 + 6 alphanumeric (branch code)
Example: SBIN0001234, HDFC0000123
Length: 11 characters
"""

import re


class IFSCValidator:
    """Validates IFSC codes."""
    
    # Format: 4 letters + 0 + 6 alphanumeric
    IFSC_PATTERN = re.compile(r'^[A-Z]{4}0[0-9]{6}$')
    
    @classmethod
    def validate(cls, ifsc: str) -> bool:
        """
        Validates an IFSC code.
        
        Args:
            ifsc: IFSC code string
            
        Returns:
            True if valid, False otherwise
        """
        if not ifsc:
            return False
        
        # Normalize (uppercase, remove spaces)
        clean = ifsc.upper().replace(' ', '').replace('-', '')
        
        # Must be exactly 11 characters
        if len(clean) != 11:
            return False
        
        # Check pattern
        if not cls.IFSC_PATTERN.match(clean):
            return False
        
        # 5th character MUST be 0
        if clean[4] != '0':
            return False
        
        return True


def validate_ifsc(ifsc: str) -> bool:
    """
    Validates an IFSC code.
    
    Args:
        ifsc: IFSC code string
        
    Returns:
        True if valid, False otherwise
    """
    return IFSCValidator.validate(ifsc)


if __name__ == "__main__":
    print("=== IFSC Code Validator Tests ===\n")
    
    test_cases = [
        ("SBIN0001234", True, "Valid SBI IFSC"),
        ("HDFC0000123", True, "Valid HDFC IFSC"),
        ("ICIC0001234", True, "Valid ICICI IFSC"),
        ("sbin0001234", True, "Valid (lowercase converted)"),
        ("SBIN 0001234", True, "Valid with space"),
        ("SBIN1001234", False, "5th char not 0"),
        ("SBI00001234", False, "Only 3 letters"),
        ("SBIN000123", False, "Too short"),
        ("SBIN00012345", False, "Too long"),
        ("1234000123A", False, "Starts with number"),
    ]
    
    for ifsc, expected, description in test_cases:
        result = validate_ifsc(ifsc)
        status = "✓" if result == expected else "✗"
        print(f"{status} {description}: {ifsc}")
        print(f"   Expected: {expected}, Got: {result}\n")
