"""
Indian Voter ID Validator
==========================
Validates Indian Voter ID (EPIC - Electors Photo Identity Card).

Format: 3 letters + 7 digits
Example: ABC1234567, XYZ9876543
"""

import re


class VoterIDValidator:
    """Validates Indian Voter IDs (EPIC cards)."""
    
    # Format: 3 letters + 7 digits
    VOTER_ID_PATTERN = re.compile(r'^[A-Z]{3}[0-9]{7}$')
    
    @classmethod
    def validate(cls, voter_id: str) -> bool:
        """
        Validates an Indian Voter ID.
        
        Args:
            voter_id: Voter ID string
            
        Returns:
            True if valid, False otherwise
        """
        if not voter_id:
            return False
        
        # Normalize (uppercase, remove spaces)
        clean = voter_id.upper().replace(' ', '').replace('-', '').replace('/', '')
        
        # Must be exactly 10 characters
        if len(clean) != 10:
            return False
        
        # Check pattern
        if not cls.VOTER_ID_PATTERN.match(clean):
            return False
        
        return True


def validate_voter_id(voter_id: str) -> bool:
    """
    Validates an Indian Voter ID.
    
    Args:
        voter_id: Voter ID string
        
    Returns:
        True if valid, False otherwise
    """
    return VoterIDValidator.validate(voter_id)


if __name__ == "__main__":
    print("=== Voter ID Validator Tests ===\n")
    
    test_cases = [
        ("ABC1234567", True, "Valid Voter ID"),
        ("XYZ9876543", True, "Valid Voter ID with XYZ"),
        ("MNO5432100", True, "Valid Voter ID with MNO"),
        ("abc1234567", True, "Valid (lowercase converted)"),
        ("ABC 1234567", True, "Valid with space"),
        ("AB1234567", False, "Only 2 letters"),
        ("ABCD123456", False, "4 letters"),
        ("ABC123456", False, "Too few digits"),
        ("ABC12345678", False, "Too many digits"),
        ("1BC1234567", False, "Starts with digit"),
    ]
    
    for voter_id, expected, description in test_cases:
        result = validate_voter_id(voter_id)
        status = "✓" if result == expected else "✗"
        print(f"{status} {description}: {voter_id}")
        print(f"   Expected: {expected}, Got: {result}\n")
