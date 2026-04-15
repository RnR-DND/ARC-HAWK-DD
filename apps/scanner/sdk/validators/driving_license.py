"""
Indian Driving License Validator
================================
Robust validation using:
- normalization
- structural parsing
- field validation
"""

import re


class DrivingLicenseValidator:
    VALID_STATE_CODES = {
        'AN', 'AP', 'AR', 'AS', 'BR', 'CH', 'CG', 'DD', 'DL', 'GA',
        'GJ', 'HP', 'HR', 'JH', 'JK', 'KA', 'KL', 'LA', 'LD', 'MH',
        'ML', 'MN', 'MP', 'MZ', 'NL', 'OD', 'OR', 'PB', 'PY', 'RJ',
        'SK', 'TN', 'TR', 'TS', 'UK', 'UP', 'WB'
    }

    @classmethod
    def validate(cls, dl: str) -> bool:
        if not dl:
            return False

        # ---------------- NORMALIZE ----------------
        clean = dl.upper().strip()

        # Remove separators
        clean = re.sub(r'[-\s]', '', clean)

        # ---------------- BASIC STRUCTURE ----------------
        # Must start with 2 letters
        if len(clean) < 14:
            return False

        if not clean[:2].isalpha():
            return False

        state_code = clean[:2]

        if state_code not in cls.VALID_STATE_CODES:
            return False

        # ---------------- EXTRACT COMPONENTS ----------------
        rest = clean[2:]

        # RTO: 1–2 digits
        if not rest[0].isdigit():
            return False

        # ---------------- YEAR-FIRST PARSING ----------------
        # Find a valid 4-digit year anywhere in rest

        for i in range(1, 4):  # possible year start positions
            if i + 4 > len(rest):
                continue

            year_str = rest[i:i+4]

            if not year_str.isdigit():
                continue

            year = int(year_str)

            if not (1900 <= year <= 2035):
                continue

            # infer RTO from prefix
            rto = rest[:i]
            serial = rest[i+4:]

            # basic checks
            if not rto.isdigit():
                continue

            if not serial.isdigit():
                continue

            if len(serial) < 5 or len(serial) > 8:
                continue

            return True

def validate_driving_license(dl: str) -> bool:
    return DrivingLicenseValidator.validate(dl)


# ---------------- TEST BLOCK ----------------
if __name__ == "__main__":
    print("=== Driving License Validator Tests ===\n")

    test_cases = [
        ("MH0120150001234", True, "Valid Maharashtra"),
        ("DL0720180005678", True, "Valid Delhi"),
        ("KA1220190009876", True, "Valid Karnataka"),
        ("mh0120150001234", True, "Lowercase"),
        ("MH-0120150001234", True, "Hyphen"),
        ("MH 0120150001234", True, "Space"),

        ("M0120150001234", False, "Invalid state"),
        ("MH012015000123", False, "Too short"),
        ("MH01201500012345", False, "Too long serial"),
        ("1H0120150001234", False, "Starts with digit"),
        ("MH0120500001234", False, "Invalid year"),
    ]

    for dl, expected, description in test_cases:
        result = validate_driving_license(dl)
        status = "✓" if result == expected else "✗"
        print(f"{status} {description}: {dl}")
        print(f"   Expected: {expected}, Got: {result}\n")
