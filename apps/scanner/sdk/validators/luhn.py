"""
Luhn Algorithm - Credit Card Validator
=======================================
The Luhn algorithm (mod 10) is used to validate credit card numbers.
"""


class Luhn:
    """Luhn algorithm implementation for credit card validation."""
    
    @classmethod
    def validate(cls, number_str: str) -> bool:
        """
        Validates a number using Luhn algorithm.
        
        Args:
            number_str: String of digits to validate
            
        Returns:
            True if checksum is valid, False otherwise
        """
        if not number_str or not number_str.isdigit():
            return False
        
        # Must be at least 13 digits for credit cards
        if len(number_str) < 13:
            return False
        
        total = 0
        parity = len(number_str) % 2
        
        for i, digit in enumerate(number_str):
            d = int(digit)
            
            # Double every second digit
            if i % 2 == parity:
                d *= 2
                if d > 9:
                    d -= 9
            
            total += d
        
        # Valid if sum is divisible by 10
        return total % 10 == 0
    
    @classmethod
    def generate_check_digit(cls, number_str: str) -> str:
        """
        Generates Luhn check digit for a number.
        
        Args:
            number_str: String of digits (without check digit)
            
        Returns:
            The check digit as a string
        """
        if not number_str or not number_str.isdigit():
            raise ValueError("Input must be a string of digits")
        
        total = 0
        parity = (len(number_str) + 1) % 2
        
        for i, digit in enumerate(number_str):
            d = int(digit)
            if i % 2 == parity:
                d *= 2
                if d > 9:
                    d -= 9
            total += d
        
        check_digit = (10 - (total % 10)) % 10
        return str(check_digit)


def validate_credit_card(number: str) -> bool:
    """
    Validates a credit card number.
    
    Args:
        number: Credit card number (may contain spaces/hyphens)
        
    Returns:
        True if valid, False otherwise
    """
    # Clean the number
    clean = ''.join(c for c in number if c.isdigit())
    
    # Must be 13-19 digits
    if len(clean) < 13 or len(clean) > 19:
        return False
        
    # Reject all zeros
    if clean == '0' * len(clean):
        return False
    
    return Luhn.validate(clean)


if __name__ == "__main__":
    print("=== Luhn Algorithm Tests ===\n")
    
    test_cases = [
        ("4532015112830366", True, "Valid Visa"),
        ("6011514433546201", True, "Valid Discover"),
        ("4532015112830367", False, "Invalid checksum"),
        ("1111111111111111", False, "Repeating sequence"),
    ]
    
    for number, expected, description in test_cases:
        result = Luhn.validate(number)
        status = "✓" if result == expected else "✗"
        print(f"{status} {description}: {number}")
        print(f"   Expected: {expected}, Got: {result}\n")
