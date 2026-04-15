"""
Luhn Algorithm - Credit Card Validator
======================================
Production-grade credit card validation using:
- Normalization
- Issuer (BIN) validation
- Luhn checksum
"""


class Luhn:
    """Luhn algorithm implementation for credit card validation."""

    @classmethod
    def validate(cls, number_str: str) -> bool:
        if not number_str or not number_str.isdigit():
            return False

        if len(number_str) < 13:
            return False

        total = 0
        parity = len(number_str) % 2

        for i, digit in enumerate(number_str):
            d = int(digit)

            if i % 2 == parity:
                d *= 2
                if d > 9:
                    d -= 9

            total += d

        return total % 10 == 0

    @classmethod
    def generate_check_digit(cls, number_str: str) -> str:
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


def valid_issuer(clean: str) -> bool:
    """
    Validates card issuer using BIN ranges.
    Covers Visa, Mastercard, Amex, Discover.
    """

    length = len(clean)

    # ---------------- VISA ----------------
    if clean.startswith('4') and length in [13, 16, 19]:
        return True

    # ------------- MASTERCARD -------------
    first2 = int(clean[:2])
    if 51 <= first2 <= 55 and length == 16:
        return True

    first4 = int(clean[:4])
    if 2221 <= first4 <= 2720 and length == 16:
        return True

    # ---------------- AMEX ----------------
    if clean.startswith(('34', '37')) and length == 15:
        return True

    # -------------- DISCOVER --------------
    if clean.startswith('6011') and length == 16:
        return True

    if clean.startswith('65') and length == 16:
        return True

    first3 = int(clean[:3])
    if 644 <= first3 <= 649 and length == 16:
        return True

    return False


def validate_credit_card(number: str) -> bool:
    """
    Validates a credit card number using:
    - normalization
    - issuer rules
    - Luhn checksum
    """

    # ---------------- NORMALIZE ----------------
    clean = ''.join(c for c in number if c.isdigit())

    # ---------------- LENGTH CHECK ----------------
    if not (13 <= len(clean) <= 19):
        return False

    # ---------------- HARD REJECTIONS ----------------
    if clean == '0' * len(clean):
        return False

    # Reject fully repeated digits (extreme garbage only)
    if len(set(clean)) == 1:
        return False

    # ---------------- ISSUER CHECK ----------------
    if not valid_issuer(clean):
        return False

    # ---------------- LUHN CHECK ----------------
    if not Luhn.validate(clean):
        return False

    return True


# ---------------- TEST BLOCK ----------------
if __name__ == "__main__":
    print("=== Credit Card Validator Tests ===\n")

    test_cases = [
        ("4532015112830366", True, "Valid Visa"),
        ("6011514433546201", True, "Valid Discover"),
        ("5555555555554444", True, "Valid Mastercard"),
        ("4000000000000002", True, "Stripe Visa test"),
        ("378282246310005", True, "Valid Amex"),

        ("4532015112830367", False, "Invalid checksum"),
        ("0000000000000000", False, "All zeros"),
        ("1111111111111111", False, "Repeating digits"),
        ("1234567890123456", False, "Invalid pattern"),
        ("9999999999999999", False, "Garbage"),
    ]

    for number, expected, description in test_cases:
        result = validate_credit_card(number)
        status = "✓" if result == expected else "✗"
        print(f"{status} {description}: {number}")
        print(f"   Expected: {expected}, Got: {result}\n")

