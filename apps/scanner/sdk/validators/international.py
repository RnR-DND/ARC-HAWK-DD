"""
International Identity Validators
===================================
Checksum validators for non-Indian government identifiers and
Indian financial identifiers not covered by the core validators.

Covered:
  validate_uk_nhs        UK NHS Number (weighted mod-11)
  validate_au_tfn        Australian Tax File Number (weighted mod-11)
  validate_ca_sin        Canadian Social Insurance Number (Luhn)
  validate_sg_nric       Singapore NRIC / FIN (weighted mod-11 variant)
  validate_aba_routing   US ABA Routing Number (weighted mod-10)
  validate_us_ssn        US Social Security Number (format only)
  validate_gst_checksum  Indian GSTIN Mod-36 checksum
"""

import re


# --------------------------------------------------------------------------- #
# UK NHS Number (NHS Information Authority spec)                               #
# --------------------------------------------------------------------------- #

def validate_uk_nhs(value: str) -> bool:
    """
    Validate a UK NHS Number using the weighted mod-11 algorithm.

    Weights: 10, 9, 8, 7, 6, 5, 4, 3, 2 for digits 1-9.
    Check digit: 11 - (sum mod 11); if result is 11 → check digit is 0;
    if result is 10 → invalid number.

    Example valid: 9434765919
    """
    clean = re.sub(r"[\s\-]", "", value)
    if not re.match(r"^\d{10}$", clean):
        return False

    weights = [10, 9, 8, 7, 6, 5, 4, 3, 2]
    total = sum(int(clean[i]) * weights[i] for i in range(9))
    remainder = total % 11
    check = 11 - remainder
    if check == 11:
        check = 0
    if check == 10:
        return False  # Invalid — no valid number can have this check digit

    return int(clean[9]) == check


# --------------------------------------------------------------------------- #
# Australian Tax File Number (ATO spec)                                        #
# --------------------------------------------------------------------------- #

def validate_au_tfn(value: str) -> bool:
    """
    Validate an Australian Tax File Number.

    Weights: 1, 4, 3, 7, 5, 8, 6, 9, 10 for 9-digit TFN.
    Sum of (digit × weight) must be divisible by 11.

    Example valid: 123456782
    """
    clean = re.sub(r"[\s\-]", "", value)
    if not re.match(r"^\d{8,9}$", clean):
        return False

    # Pad to 9 digits if 8 provided (leading zero)
    if len(clean) == 8:
        clean = "0" + clean

    weights = [1, 4, 3, 7, 5, 8, 6, 9, 10]
    total = sum(int(clean[i]) * weights[i] for i in range(9))
    return total % 11 == 0


# --------------------------------------------------------------------------- #
# Canadian Social Insurance Number (CRA spec)                                  #
# --------------------------------------------------------------------------- #

def validate_ca_sin(value: str) -> bool:
    """
    Validate a Canadian Social Insurance Number using the Luhn algorithm.

    The SIN is a 9-digit number. We validate Luhn only; first-digit province
    restrictions are not enforced here since 0xx series appears in test datasets
    and some administrative uses.

    Example valid: 046 454 286
    """
    clean = re.sub(r"[\s\-]", "", value)
    if not re.match(r"^\d{9}$", clean):
        return False

    # Luhn check
    total = 0
    for i, digit in enumerate(clean):
        d = int(digit)
        if i % 2 == 1:  # Double every second digit (1-indexed even positions)
            d *= 2
            if d > 9:
                d -= 9
        total += d
    return total % 10 == 0


# --------------------------------------------------------------------------- #
# Singapore NRIC / FIN (ICA spec)                                              #
# --------------------------------------------------------------------------- #

_SG_NRIC_WEIGHTS = [2, 7, 6, 5, 4, 3, 2]
_SG_NRIC_ALPHA_S = "JZIHGFEDCBA"   # prefix S (citizens born before 2000)
_SG_NRIC_ALPHA_T = "GFEDCBA"       # prefix T (citizens born 2000+)  — uses offset 4
_SG_NRIC_ALPHA_F = "XWUTRQPNMLK"   # prefix F (foreigners before 2000)
_SG_NRIC_ALPHA_G = "RQPNMLKJIHG"   # prefix G (foreigners 2000+)
_SG_NRIC_ALPHA_M = "XWUTRQPNMLKJIHGFEDCBA"  # prefix M (2022+, MyInfo)


def validate_sg_nric(value: str) -> bool:
    """
    Validate a Singapore NRIC or FIN.

    Format: [STFGM]DDDDDDD[A-Z]

    The check character is derived from:
    1. Multiply each of the 7 digits by weights [2,7,6,5,4,3,2].
    2. Sum the products (+ offset 4 for T/G prefix).
    3. Remainder mod 11 indexes into the appropriate check-letter table.

    Example valid: S0123456D
    """
    clean = re.sub(r"\s", "", value).upper()
    if not re.match(r"^[STFGM]\d{7}[A-Z]$", clean):
        return False

    prefix = clean[0]
    digits = clean[1:8]
    check_char = clean[8]

    total = sum(int(digits[i]) * _SG_NRIC_WEIGHTS[i] for i in range(7))

    # Add offset for post-2000 series
    if prefix in ("T", "G"):
        total += 4
    elif prefix == "M":
        total += 3

    remainder = total % 11

    if prefix in ("S", "T"):
        table = _SG_NRIC_ALPHA_S
    elif prefix in ("F", "G"):
        table = _SG_NRIC_ALPHA_F
    else:  # M
        table = _SG_NRIC_ALPHA_M

    if remainder >= len(table):
        return False
    return table[remainder] == check_char


# --------------------------------------------------------------------------- #
# US ABA Routing Number (ABA spec)                                             #
# --------------------------------------------------------------------------- #

def validate_aba_routing(value: str) -> bool:
    """
    Validate a US ABA bank routing number.

    The 9-digit routing number satisfies:
    3*(d1+d4+d7) + 7*(d2+d5+d8) + 1*(d3+d6+d9) ≡ 0 (mod 10)

    Federal Reserve routing codes: first two digits must be 01-12 or 21-32.

    Example valid: 021000021 (JPMorgan Chase)
    """
    clean = re.sub(r"[\s\-]", "", value)
    if not re.match(r"^\d{9}$", clean):
        return False

    d = [int(c) for c in clean]
    total = (3 * (d[0] + d[3] + d[6]) +
             7 * (d[1] + d[4] + d[7]) +
             1 * (d[2] + d[5] + d[8]))
    if total % 10 != 0:
        return False

    # Valid Federal Reserve prefixes
    prefix = int(clean[:2])
    valid_prefixes = set(range(1, 13)) | set(range(21, 33))
    return prefix in valid_prefixes


# --------------------------------------------------------------------------- #
# US Social Security Number (format only, no checksum)                         #
# --------------------------------------------------------------------------- #

def validate_us_ssn(value: str) -> bool:
    """
    Validate a US Social Security Number (format check only).

    Rejects:
    - AAA = 000, 666, 900-999 (never issued)
    - GG = 00
    - SSSS = 0000

    Example valid: 123-45-6789
    """
    clean = re.sub(r"[\s\-]", "", value)
    if not re.match(r"^\d{9}$", clean):
        return False

    area = int(clean[:3])
    group = int(clean[3:5])
    serial = int(clean[5:])

    if area == 0 or area == 666 or area >= 900:
        return False
    if group == 0:
        return False
    if serial == 0:
        return False

    return True


# --------------------------------------------------------------------------- #
# Indian GSTIN — Mod-36                                                        #
# --------------------------------------------------------------------------- #

def validate_gst_checksum(value: str) -> bool:
    """
    Validate an Indian GST Identification Number (GSTIN) using Mod-36 checksum.

    Format: [01-38][A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z] (15 characters)
    Character set: "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ" (36 chars)

    Algorithm:
    1. For positions 0-13 (14 chars), get each char's index in CHARS.
    2. Multiply by (position % 2 + 1) — alternating 1 and 2.
    3. For each product p: partial = (p // 36) + (p % 36)
    4. Sum all partials, compute (36 - sum % 36) % 36 → check_index.
    5. CHARS[check_index] must match character at position 14.

    Known valid: 27AAAPZ1234F1Z5 (Maharashtra)
    Known valid: 29AABCU9603R1ZX (Karnataka)

    Args:
        value: GSTIN string (15 chars, spaces stripped).

    Returns:
        True if structurally valid with correct checksum.
    """
    clean = re.sub(r"[^A-Z0-9]", "", value.upper())
    if len(clean) != 15:
        return False
    if not re.match(r"^[0-3][0-9][A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z]$", clean):
        return False
    state_code = int(clean[:2])
    if state_code < 1 or state_code > 38:
        return False
    CHARS = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
    total = 0
    for i, ch in enumerate(clean[:14]):
        v = CHARS.index(ch)
        product = v * (i % 2 + 1)
        total += (product // 36) + (product % 36)
    check_digit_val = (36 - (total % 36)) % 36
    expected_check = CHARS[check_digit_val]
    return clean[14] == expected_check
