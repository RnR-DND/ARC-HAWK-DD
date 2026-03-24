"""
Verhoeff Algorithm Implementation
----------------------------------
The Verhoeff algorithm is a checksum formula for error detection developed by Dutch mathematician Jacobus Verhoeff.
It is used to validate Aadhaar numbers (India's 12-digit unique identification number).

Algorithm:
1. Uses a dihedral group D5 (multiplication table)
2. Uses a permutation table
3. Uses an inverse table
4. The check digit is the value that makes the final checksum 0

Reference: https://en.wikipedia.org/wiki/Verhoeff_algorithm
"""

# Multiplication table (dihedral group D5)
MULTIPLICATION_TABLE = [
    [0, 1, 2, 3, 4, 5, 6, 7, 8, 9],
    [1, 2, 3, 4, 0, 6, 7, 8, 9, 5],
    [2, 3, 4, 0, 1, 7, 8, 9, 5, 6],
    [3, 4, 0, 1, 2, 8, 9, 5, 6, 7],
    [4, 0, 1, 2, 3, 9, 5, 6, 7, 8],
    [5, 9, 8, 7, 6, 0, 4, 3, 2, 1],
    [6, 5, 9, 8, 7, 1, 0, 4, 3, 2],
    [7, 6, 5, 9, 8, 2, 1, 0, 4, 3],
    [8, 7, 6, 5, 9, 3, 2, 1, 0, 4],
    [9, 8, 7, 6, 5, 4, 3, 2, 1, 0]
]

# Permutation table
PERMUTATION_TABLE = [
    [0, 1, 2, 3, 4, 5, 6, 7, 8, 9],
    [1, 5, 7, 6, 2, 8, 3, 0, 9, 4],
    [5, 8, 0, 3, 7, 9, 6, 1, 4, 2],
    [8, 9, 1, 6, 0, 4, 3, 5, 2, 7],
    [9, 4, 5, 3, 1, 2, 6, 8, 7, 0],
    [4, 2, 8, 6, 5, 7, 3, 9, 0, 1],
    [2, 7, 9, 3, 8, 0, 6, 4, 1, 5],
    [7, 0, 4, 6, 9, 1, 3, 2, 5, 8]
]

# Inverse table
INVERSE_TABLE = [0, 4, 3, 2, 1, 5, 6, 7, 8, 9]


def verhoeff_validate(number: str) -> bool:
    """
    Validates a number using the Verhoeff algorithm.
    
    Args:
        number: String of digits to validate
        
    Returns:
        True if valid, False otherwise
    """
    # Input validation
    if not number:
        return False
    
    if not number.isdigit():
        return False
    
    # Verhoeff checksum
    checksum = 0
    
    # Process digits from right to left
    for i, digit in enumerate(reversed(number)):
        digit_value = int(digit)
        
        # Get the permutation based on position
        # Position is (i % 8) because permutation table has 8 rows
        permutation_row = i % 8
        permuted_digit = PERMUTATION_TABLE[permutation_row][digit_value]
        
        # Apply multiplication table
        checksum = MULTIPLICATION_TABLE[checksum][permuted_digit]
    
    # Valid if checksum is 0
    return checksum == 0


def verhoeff_generate_check_digit(number: str) -> str:
    """
    Generates the Verhoeff check digit for a given number.
    
    Args:
        number: String of digits (without check digit)
        
    Returns:
        The check digit as a string
    """
    if not number or not number.isdigit():
        raise ValueError("Input must be a string of digits")
    
    # Calculate checksum with check digit position
    checksum = 0
    
    # Process from right to left, but position starts at 1 because we're adding a digit
    for i, digit in enumerate(reversed(number)):
        digit_value = int(digit)
        position = (i + 1) % 8  # +1 because we're inserting a check digit at position 0
        permuted_digit = PERMUTATION_TABLE[position][digit_value]
        checksum = MULTIPLICATION_TABLE[checksum][permuted_digit]
    
    # The check digit is the inverse of the checksum
    check_digit = INVERSE_TABLE[checksum]
    
    return str(check_digit)


# Example usage and tests
if __name__ == "__main__":
    # Test with known valid Aadhaar-like numbers
    
    # Valid 12-digit number with Verhoeff check digit
    test_valid = "123456789012"
    print(f"Testing {test_valid}: {verhoeff_validate(test_valid)}")
    
    # Generate check digit for 11 digits
    base_number = "12345678901"
    check_digit = verhoeff_generate_check_digit(base_number)
    full_number = base_number + check_digit
    print(f"Generated number: {full_number}, Valid: {verhoeff_validate(full_number)}")
    
    # Test invalid numbers
    invalid1 = "123456789013"  # Wrong check digit
    invalid2 = "111111111111"  # Repetition
    invalid3 = "123456789abc"  # Non-numeric
    
    print(f"Invalid test 1: {verhoeff_validate(invalid1)}")  # Should be False
    print(f"Invalid test 2: {verhoeff_validate(invalid2)}")  # Should be False
    print(f"Invalid test 3: {verhoeff_validate(invalid3)}")  # Should be False
