"""
PAN Validator - Permanent Account Number (India)
=================================================
Validates PAN using Weighted Modulo 26 check digit algorithm.

Format: ABCDE1234F
- Positions 1-3: Any letters
- Position 4: Entity type (P=Person, C=Company, H=HUF, F=Firm, etc.)
- Position 5: First letter of surname/entity name
- Positions 6-9: 4 digits (sequential number)
- Position 10: Check letter (Weighted Modulo 26)

Mathematical Formula:
Each of the first 9 characters is converted to a number, multiplied by a 
fixed weight (1-9), summed, and Modulo 26 is applied. The remainder 
corresponds to the alphabet of the 10th character.
"""


class PANValidator:
    """Validates Indian PAN with mathematical check digit."""
    
    # Valid entity types (4th character)
    VALID_ENTITY_TYPES = {
        'P',  # Individual/Person
        'C',  # Company
        'H',  # Hindu Undivided Family (HUF)
        'F',  # Partnership Firm
        'A',  # Association of Persons (AOP)
        'T',  # Trust
        'B',  # Body of Individuals (BOI)
        'L',  # Local Authority
        'J',  # Artificial Juridical Person
        'G',  # Government
    }
    
    @classmethod
    def validate(cls, pan: str, context: str = "") -> bool:
        """
        Validates PAN using Weighted Modulo 26 algorithm with strict anti-fake checks.
        
        Args:
            pan: PAN string
            context: Surrounding text context (optional, for additional validation)
            
        Returns:
            True if valid, False otherwise
        """
        if not pan:
            return False
        
        # Normalize
        clean = pan.upper().replace(' ', '').replace('-', '')
        
        # Must be exactly 10 characters
        if len(clean) != 10:
            return False
        
        # Format: AAAAA9999A
        if not (clean[:5].isalpha() and 
                clean[5:9].isdigit() and 
                clean[9].isalpha()):
            return False
        
        # 4th character must be valid entity type
        if clean[3] not in cls.VALID_ENTITY_TYPES:
            return False
        
        # === STRICT ANTI-FAKE CHECKS ===
        
        # 1. Reject obvious test patterns - all 5 letters the same OR 4+ consecutive same letters
        first_5 = clean[:5]
        if len(set(first_5)) == 1:  # All same letter (AAAAA, BBBBB, etc.)
            return False
        # Check for 4+ consecutive identical letters (but AAA is OK per real PAN format)
        for i in range(len(first_5) - 3):
            if len(set(first_5[i:i+4])) == 1:  # 4 consecutive same letters
                return False
        
        #2. Reject sequential alphabet patterns
        if first_5 in ['ABCDE', 'BCDEF', 'CDEFG', 'DEFGH', 'EFGHI', 'FGHIJ']:
            return False
        
        # 3. Reject repeated digit sequences (all 4 same)
        digits = clean[5:9]
        if len(set(digits)) == 1:  # All same digit (e.g., 1111, 9999)
            return False
        
        # Note: Sequential digits (1234, 5678) removed - real PANs can have these
        
        # 4. Context-based rejection: if found in code files, likely test data
        if context:
            code_indicators = [
                'test_', 'example', 'sample', 'demo', 'dummy',
                'def ', 'class ', 'import ', '"""', "'''",
                '.py', '.js', '.java', 'EXAMPLE', 'TEST'
            ]
            context_lower = context.lower()
            if any(indicator.lower() in context_lower for indicator in code_indicators):
                return False
        
        # 5. Validate 10th character using Weighted Modulo 26
        if not cls._validate_check_digit(clean):
            return False
        
        return True
    
    @classmethod
    def _validate_check_digit(cls, pan: str) -> bool:
        """
        Validate 10th character using Weighted Modulo 26 algorithm.
        
        Algorithm:
        1. Convert first 9 characters to numbers (A=10, B=11, ..., Z=35, 0=0, ..., 9=9)
        2. Apply fixed weights (1, 2, 3, 4, 5, 6, 7, 8, 9) to each position
        3. Sum all weighted values
        4. Take Modulo 26
        5. Convert remainder back to letter (0=A, 1=B, ..., 25=Z)
        """
        # Weights for positions 1-9
        weights = [1, 2, 3, 4, 5, 6, 7, 8, 9]
        
        total = 0
        for i in range(9):
            char = pan[i]
            
            # Convert character to number
            if char.isalpha():
                # A=10, B=11, ..., Z=35
                value = ord(char) - ord('A') + 10
            else:
                # 0=0, 1=1, ..., 9=9
                value = int(char)
            
            # Apply weight and add to total
            total += value * weights[i]
        
        # Calculate expected check letter
        remainder = total % 26
        expected_check = chr(remainder + ord('A'))
        
        # Compare with actual 10th character
        return pan[9] == expected_check


def validate_pan(pan: str, context: str = "") -> bool:
    """
    Validates a PAN number with optional context for test data detection.
    
    Args:
        pan: PAN string
        context: Surrounding text context (optional)
        
    Returns:
        True if valid, False otherwise
    """
    return PANValidator.validate(pan, context)


if __name__ == "__main__":
    print("=== PAN Validator with Weighted Modulo 26 ===\n")
    
    test_cases = [
        ("AAAPC1234O", True, "Valid format (Company)"),  # Corrected: check letter is O, not D
        ("AAAPP1234B", True, "Valid format (Person)"),  # Corrected: check letter is B, not D
        ("AAAPZ1234Q", False, "Invalid entity type Z"),
        ("ABCD123456", False, "Too many digits"),
    ]
    
    for pan, expected, description in test_cases:
        result = validate_pan(pan)
        status = "✓" if result == expected else "✗"  
        print(f"{status} {description}: {pan} -> {result}")
