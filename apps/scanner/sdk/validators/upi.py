"""
UPI ID Validator
================
Robust validation for UPI IDs.
"""

import re


class UPIValidator:
    # Expanded known providers (not exhaustive)
    KNOWN_PROVIDERS = {
        'paytm', 'phonepe', 'googlepay', 'gpay', 'ybl',
        'oksbi', 'okhdfcbank', 'okaxis', 'okicici',
        'ibl', 'airtel', 'fbl', 'pockets', 'apl',
        'upi', 'axl'
    }

    EMAIL_DOMAINS = {
        "gmail", "yahoo", "outlook", "hotmail", "protonmail",
        "icloud", "live", "msn"
    }

    # More flexible pattern
    UPI_PATTERN = re.compile(r'^[a-z0-9._-]+@[a-z0-9]+$')

    @classmethod
    def validate(cls, upi: str) -> bool:
        if not upi:
            return False

        # ---------------- NORMALIZE ----------------
        upi = upi.strip().lower()

        # Remove trailing punctuation (real-world logs)
        upi = re.sub(r'[^\w@._-]+$', '', upi)

        # Must contain exactly one @
        if upi.count('@') != 1:
            return False

        # Basic pattern
        if not cls.UPI_PATTERN.match(upi):
            return False

        if len(upi) > 50:  # relaxed upper bound
            return False

        user, provider = upi.split('@')

        # ---------------- USER VALIDATION ----------------
        if not (3 <= len(user) <= 30):
            return False

        # Avoid extreme garbage only
        if len(set(user)) == 1:
            return False

        # ---------------- PROVIDER VALIDATION ----------------
        if not (2 <= len(provider) <= 25):
            return False

        # Known providers → strong signal
        if provider in cls.KNOWN_PROVIDERS:
            return True

        # Common UPI pattern: ok*
        if provider.startswith("ok") and provider.isalnum():
            return True

        if provider in cls.EMAIL_DOMAINS:
            return False

        return False


def validate_upi(upi: str) -> bool:
    return UPIValidator.validate(upi)


# ---------------- TEST BLOCK ----------------
if __name__ == "__main__":
    print("=== UPI ID Validator Tests ===\n")

    test_cases = [
        ("user@paytm", True),
        ("9876543210@ybl", True),
        ("john.doe@phonepe", True),
        ("user_name@googlepay", True),
        ("test-user@oksbi", True),
        ("rahul@okhdfcbank", True),
        ("rahul123@upi", True),

        ("invalid", False),
        ("@paytm", False),
        ("user@", False),
        ("user@@paytm", False),

        ("a@upi", False),  # too short user
        ("user@x", False),  # too short provider

        ("aaaaa@upi", False),  # extreme repetition
    ]

    for upi, expected in test_cases:
        result = validate_upi(upi)
        status = "✓" if result == expected else "✗"
        print(f"{status} {upi} → {result} (expected {expected})")
