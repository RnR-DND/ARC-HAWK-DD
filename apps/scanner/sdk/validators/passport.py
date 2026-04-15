"""
Indian Passport Number Validator
================================
Strict validation based on:
- Format (1 letter + 7 digits)
- Valid prefix categories
"""

import re


class IndianPassportValidator:
    # Passport types
    DIPLOMATIC_TYPES = {'J', 'Z'}
    PERSONAL_TYPES = set('ABCDEFGHIKLMNOPQRSTUVWX')

    VALID_PREFIXES = DIPLOMATIC_TYPES | PERSONAL_TYPES

    PASSPORT_PATTERN = re.compile(r'^[A-Z][0-9]{7}$')

    @classmethod
    def validate(cls, passport: str) -> bool:
        if not passport:
            return False

        # ---------------- NORMALIZE ----------------
        clean = passport.upper().strip()
        clean = re.sub(r'[^A-Z0-9]', '', clean)

        # ---------------- STRUCTURE CHECK ----------------
        if not cls.PASSPORT_PATTERN.fullmatch(clean):
            return False

        # ---------------- PREFIX CHECK ----------------
        if clean[0] not in cls.VALID_PREFIXES:
            return False

        return True

    @classmethod
    def extract(cls, text: str) -> str | None:
        if not text:
            return None

        match = re.search(r'[A-Z][0-9]{7}', text.upper())
        return match.group() if match else None


def validate_indian_passport(passport: str) -> bool:
    return IndianPassportValidator.validate(passport)


# ---------------- TEST BLOCK ----------------
if __name__ == "__main__":
    print("=== Indian Passport Validator Tests ===\n")

    test_cases = [
        ("A1234567", True),
        ("Z9876543", True),
        ("M5432109", True),
        ("a1234567", True),
        ("A 1234567", True),

        ("12345678", False),
        ("AB123456", False),
        ("A12345", False),
        ("A123456789", False),

        ("Q1234567", False),  # invalid prefix
        ("X1234567", False),  # invalid prefix
    ]

    for passport, expected in test_cases:
        result = validate_indian_passport(passport)
        status = "✓" if result == expected else "✗"
        print(f"{status} {passport} → {result} (expected {expected})")
