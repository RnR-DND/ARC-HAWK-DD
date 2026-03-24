"""
Dummy Data Detector
===================
Detects patterns that are mathematically valid but semantically meaningless.
Examples: 111111111111, 123456789012, keyboard walks
"""

def is_dummy_data(text: str) -> bool:
    """
    Returns True if data appears to be dummy/test data.
    
    Enhanced with additional pattern detection:
    - Repeating triplets (123123123...)
    - Explicit zero check
    - Better entropy detection
    
    Args:
        text: String to check (should be digits only)
        
    Returns:
        True if data is likely dummy, False otherwise
    """
    if not text or len(text) < 3:
        return False
    
    # Check 1: All same digit (1111111..., 0000000...)
    if len(set(text)) == 1:
        return True
    
    # Check 2: Linear ascending/descending sequence (1234567..., 9876543...)
    digits = [int(d) for d in text if d.isdigit()]
    if len(digits) >= 4:
        # Check ascending (with wraparound)
        is_ascending = all(digits[i+1] == (digits[i] + 1) % 10 for i in range(len(digits)-1))
        if is_ascending:
            return True
        
        # Check descending (with wraparound)
        # Need to handle negative modulo correctly
        is_descending = all(digits[i+1] == (digits[i] - 1 + 10) % 10 for i in range(len(digits)-1))
        if is_descending:
            return True
    
    # Check 3: Repeating sequences (1212..., 123123..., 12341234...)
    if len(text) >= 6:
        for n in range(2, len(text) // 2 + 1):
            block = text[:n]
            if text == block * (len(text) // n) + block[:len(text) % n]:
                return True
    
    # Check 5: Too simple (entropy check)
    # Only apply if length is significant AND entropy is very low
    unique_digits = len(set(text))
    if len(text) >= 10 and unique_digits <= 2:
        return True
    
    # Check 6: Specific known test patterns
    # Don't flag valid-looking 10-digit phone numbers as dummy
    # Phone numbers have moderate entropy (not sequential, not repeating)
    if len(text) == 10 and unique_digits >= 4:
        # This is likely a valid phone number, not dummy data
        return False
    
    return False


if __name__ == "__main__":
    print("=== Dummy Data Detection Tests ===\n")
    
    test_cases = [
        ("111111111111", True, "All same"),
        ("000000000000", True, "All zeros"),
        ("123456789012", True, "Ascending sequence"),
        ("987654321000", True, "Descending sequence (partial)"),
        ("9876543210", True, "Descending sequence (full)"),  # This IS sequential!
        ("121212121212", True, "Repeating pairs"),
        ("123123123123", True, "Repeating triplets"),
        ("456456456456", True, "Repeating triplets (456)"),
        ("999911112222", False, "Real-looking data"),
        ("234567890126", False, "Valid Aadhaar"),
        ("9123456789", False, "Valid phone (non-sequential)"),
        ("8765432109", False, "Valid phone (not full sequence)"),
    ]
    
    for data, expected, description in test_cases:
        result = is_dummy_data(data)
        status = "✓" if result == expected else "✗"
        print(f"{status} {description}: {data}")
        print(f"   Expected: {expected}, Got: {result}\n")
