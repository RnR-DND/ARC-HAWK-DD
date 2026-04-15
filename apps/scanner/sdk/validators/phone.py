"""
Indian Phone Number Validator
==============================
Robust validation for Indian mobile numbers.
"""

import re


class IndianPhoneValidator:
    VALID_PREFIXES = {'6', '7', '8', '9'}

    PHONE_PATTERN = re.compile(r'^\d{10}$')

    @classmethod
    def validate(cls, phone: str) -> bool:
        if not phone:
            return False


        clean = cls._clean_phone(phone)

        if not clean:
            return False

        # Must be exactly 10 digits
        if not cls.PHONE_PATTERN.match(clean):
            return False

        # Must start with valid mobile prefix
        if clean[0] not in cls.VALID_PREFIXES:
            return False

        # Only reject EXTREME fake patterns
        if cls._is_obviously_fake(clean):
            return False

        if clean.startswith("60") or clean.startswith("61"):
            pass

        return True

    @staticmethod
    def _clean_phone(phone: str) -> str:
        """
        Normalize phone number:
        - remove non-digits
        - handle country codes
        """

        # Remove non-digits
        clean = re.sub(r'\D', '', phone)

        if not clean:
            return ""

        clean = clean.replace("91 ", "91")

        # Handle country code cases
        if clean.startswith('91') and len(clean) >= 12:
            clean = clean[-10:]

        # 0091XXXXXXXXXX
        if clean.startswith('0091') and len(clean) >= 14:
            clean = clean[-10:]

        # 0XXXXXXXXXX
        if clean.startswith('0') and len(clean) >= 11:
            clean = clean[-10:]

        # After normalization, must be <=10
        if len(clean) != 10:
            return ""

        return clean

    @staticmethod
    def _is_obviously_fake(phone: str) -> bool:
        """
        Only reject extremely obvious fake numbers.
        DO NOT over-restrict.
        """

        # All same digits → fake
        if len(set(phone)) == 1:
            return True

        # Strict sequential (no wraparound)
        ascending = all(
            int(phone[i]) == int(phone[i - 1]) + 1
            for i in range(1, len(phone))
        )

        descending = all(
            int(phone[i]) == int(phone[i - 1]) - 1
            for i in range(1, len(phone))
        )

        if ascending or descending:
            return True

        return False


def validate_indian_phone(phone: str) -> bool:
    return IndianPhoneValidator.validate(phone)


# ---------------- TEST BLOCK ----------------
if __name__ == "__main__":
    print("=== Indian Phone Validator Tests ===\n")

    test_cases = [
        ("9876543210", True, "Valid mobile"),
        ("8765432109", True, "Valid mobile"),
        ("7654321098", True, "Valid mobile"),
        ("6543210987", True, "Valid mobile"),

        ("+91 9876543210", True, "With +91"),
        ("91 9876543210", True, "With 91"),
        ("0091 9876543210", True, "With 0091"),
        ("09876543210", True, "Leading 0"),

        ("5876543210", False, "Invalid prefix"),
        ("987654321", False, "Too short"),
        ("98765432109", False, "Too long"),

        ("9999999999", False, "All same digits"),
        ("0123456789", False, "Sequential"),
        ("1234567890", False, "Invalid prefix"),

        ("7890123456", True, "Valid wrap sequence"),
        ("9000000001", True, "Low diversity but valid"),
    ]

    for phone, expected, description in test_cases:
        result = validate_indian_phone(phone)
        status = "✓" if result == expected else "✗"
        print(f"{status} {description}: {phone}")
        print(f"   Expected: {expected}, Got: {result}\n")
