"""
Hawk PII Validators — Consolidated Validator Library
=====================================================
All mathematical checksum and format validators used by hawk_patterns.py.

Each validator:
  - Accepts a raw string (may have spaces/hyphens/other separators)
  - Returns True if the value is structurally valid, False otherwise
  - Does NOT raise exceptions on malformed input (returns False instead)

Validators in this file:
  verhoeff_validate(n)        — Aadhaar checksum (dihedral group D5)
  luhn_validate(n)            — Credit/debit card, bank card
  pan_checksum_validate(s)    — Indian PAN card
  gstin_validate(s)           — Indian GSTIN (state code + PAN + format)
  ifsc_validate(s)            — Indian IFSC code
  mobile_india_validate(s)    — Indian mobile numbers (all formats)
  pincode_validate(s)         — Indian PIN code (110001–855117)
  aadhaar_age_plausibility(dob_str) — Cross-field DOB plausibility
  micr_validate(s)            — 9-digit MICR code
  swift_bic_validate(s)       — SWIFT/BIC code
  email_validate(s)           — RFC 5322 email
  ctri_validate(s)            — Clinical trial registration number (India)
"""

from __future__ import annotations

import re
import math
from typing import Optional


# ─────────────────────────────────────────────────────────────────────────────
# VERHOEFF (Aadhaar)
# ─────────────────────────────────────────────────────────────────────────────

_VH_D = [
    [0, 1, 2, 3, 4, 5, 6, 7, 8, 9],
    [1, 2, 3, 4, 0, 6, 7, 8, 9, 5],
    [2, 3, 4, 0, 1, 7, 8, 9, 5, 6],
    [3, 4, 0, 1, 2, 8, 9, 5, 6, 7],
    [4, 0, 1, 2, 3, 9, 5, 6, 7, 8],
    [5, 9, 8, 7, 6, 0, 4, 3, 2, 1],
    [6, 5, 9, 8, 7, 1, 0, 4, 3, 2],
    [7, 6, 5, 9, 8, 2, 1, 0, 4, 3],
    [8, 7, 6, 5, 9, 3, 2, 1, 0, 4],
    [9, 8, 7, 6, 5, 4, 3, 2, 1, 0],
]

_VH_P = [
    [0, 1, 2, 3, 4, 5, 6, 7, 8, 9],
    [1, 5, 7, 6, 2, 8, 3, 0, 9, 4],
    [5, 8, 0, 3, 7, 9, 6, 1, 4, 2],
    [8, 9, 1, 6, 0, 4, 3, 5, 2, 7],
    [9, 4, 5, 3, 1, 2, 6, 8, 7, 0],
    [4, 2, 8, 6, 5, 7, 3, 9, 0, 1],
    [2, 7, 9, 3, 8, 0, 6, 4, 1, 5],
    [7, 0, 4, 6, 9, 1, 3, 2, 5, 8],
]

_VH_INV = [0, 4, 3, 2, 1, 5, 6, 7, 8, 9]


def verhoeff_validate(number: str) -> bool:
    """
    Validate a number using the Verhoeff checksum algorithm.

    Used for: Aadhaar UID (12 digits).
    Detects all single-digit errors and adjacent transpositions.

    Args:
        number: Digit string (spaces/hyphens stripped internally).

    Returns:
        True if Verhoeff checksum is valid (checksum == 0).
    """
    clean = re.sub(r"[\s\-]", "", number)
    if not clean.isdigit():
        return False
    checksum = 0
    for i, digit in enumerate(reversed(clean)):
        checksum = _VH_D[checksum][_VH_P[i % 8][int(digit)]]
    return checksum == 0


def verhoeff_generate_check_digit(number: str) -> str:
    """Generate Verhoeff check digit for a digit string (without check digit)."""
    clean = re.sub(r"[\s\-]", "", number)
    if not clean.isdigit():
        raise ValueError("Input must be digits only")
    checksum = 0
    for i, digit in enumerate(reversed(clean)):
        checksum = _VH_D[checksum][_VH_P[(i + 1) % 8][int(digit)]]
    return str(_VH_INV[checksum])


# ─────────────────────────────────────────────────────────────────────────────
# LUHN (Credit/Debit Cards, Canadian SIN)
# ─────────────────────────────────────────────────────────────────────────────

def luhn_validate(number: str) -> bool:
    """
    Validate a number using the Luhn algorithm (ISO/IEC 7812-1).

    Used for: credit cards, debit cards, Canadian SIN.
    Detects all single-digit errors.

    Args:
        number: Digit string (spaces/hyphens stripped internally).

    Returns:
        True if Luhn checksum is valid.
    """
    clean = re.sub(r"[\s\-]", "", number)
    if not clean.isdigit() or len(clean) < 2:
        return False
    total = 0
    parity = len(clean) % 2
    for i, digit in enumerate(clean):
        d = int(digit)
        if i % 2 == parity:
            d *= 2
            if d > 9:
                d -= 9
        total += d
    return total % 10 == 0


# ─────────────────────────────────────────────────────────────────────────────
# PAN CHECKSUM
# ─────────────────────────────────────────────────────────────────────────────

def pan_checksum_validate(pan: str) -> bool:
    """
    Validate Indian PAN card format and entity-type check.

    Format: AAAAA9999A
    - Positions 1-3: City/AO code (alphabetic)
    - Position 4: Entity type (P=Person, C=Company, H=HUF, F=Firm, etc.)
    - Position 5: First letter of surname/name
    - Positions 6-9: Sequential number
    - Position 10: Alphabetic check character

    Valid entity codes: A B C F G H J L P T K E

    Args:
        pan: PAN string (case-insensitive).

    Returns:
        True if format and entity code are valid.
    """
    clean = re.sub(r"[\s]", "", pan).upper()
    if len(clean) != 10:
        return False
    if not re.match(r"^[A-Z]{5}[0-9]{4}[A-Z]$", clean):
        return False
    # Entity type validation (4th character)
    valid_entity_codes = set("ABCFGHJLPTKE")
    if clean[3] not in valid_entity_codes:
        return False
    # Reserved sequential numbers that are invalid
    if clean[5:9] == "0000":
        return False
    return True


# ─────────────────────────────────────────────────────────────────────────────
# GSTIN
# ─────────────────────────────────────────────────────────────────────────────

# Valid Indian state codes (GST)
_GSTIN_STATE_CODES = {
    1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18,
    19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34,
    35, 36, 37, 38,
}


def gstin_validate(gstin: str) -> bool:
    """
    Validate Indian GSTIN format and state code.

    Format: SS AAAAA9999A E Z C
    - SS: State code (01–38)
    - AAAAA9999A: Embedded PAN (10 chars)
    - E: Entity number (1-9 or A-Z)
    - Z: Always 'Z'
    - C: Checksum character (0-9 or A-Z)

    Args:
        gstin: GSTIN string (case-insensitive, spaces stripped).

    Returns:
        True if format and state code are valid.
    """
    clean = re.sub(r"[\s]", "", gstin).upper()
    if len(clean) != 15:
        return False
    if not re.match(r"^[0-3][0-9][A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z]$", clean):
        return False
    state_code = int(clean[:2])
    if state_code not in _GSTIN_STATE_CODES:
        return False
    # Validate embedded PAN (positions 3-12)
    embedded_pan = clean[2:12]
    return pan_checksum_validate(embedded_pan)


# ─────────────────────────────────────────────────────────────────────────────
# IFSC
# ─────────────────────────────────────────────────────────────────────────────

def ifsc_validate(ifsc: str) -> bool:
    """
    Validate Indian Financial System Code (IFSC).

    Format: BBBB0XXXXXX
    - BBBB: 4-character bank code (alpha)
    - 0: Always zero (5th character)
    - XXXXXX: 6-character branch code (alphanumeric)

    Args:
        ifsc: IFSC string (case-insensitive).

    Returns:
        True if format is valid.
    """
    clean = re.sub(r"[\s]", "", ifsc).upper()
    if len(clean) != 11:
        return False
    if not re.match(r"^[A-Z]{4}0[A-Z0-9]{6}$", clean):
        return False
    return True


# ─────────────────────────────────────────────────────────────────────────────
# INDIAN MOBILE
# ─────────────────────────────────────────────────────────────────────────────

# Valid Indian mobile operator ranges (first digit of 10-digit number)
# All Indian mobile numbers start with 6, 7, 8, or 9
_MOBILE_VALID_FIRST_DIGITS = set("6789")

# More precise: valid 2-digit prefixes (first two digits of 10-digit subscriber number)
# Ranges assigned by TRAI to operators: 70x-79x, 80x-89x, 90x-99x, 60x-69x
# Known INVALID prefixes to exclude: 700 is unassigned historically, but 70x are mostly valid now
# Use first-digit check as primary, operator-range as boost
_MOBILE_INVALID_PREFIXES = {
    "60", "61", "62", "63", "64", "65",  # Most 60x-65x unassigned
}


def mobile_india_validate(number: str) -> bool:
    """
    Validate Indian mobile number in any of the 4 standard formats:
    1. +91XXXXXXXXXX (international)
    2. 91XXXXXXXXXX (without +)
    3. 0XXXXXXXXXX (with leading 0)
    4. XXXXXXXXXX (bare 10 digits)

    Validation:
    - Normalise to 10-digit subscriber number
    - First digit must be 6-9
    - Not all same digit (e.g., 9999999999 invalid)
    - Not a known test/dummy number sequence

    Args:
        number: Phone number in any format (spaces/hyphens stripped).

    Returns:
        True if valid Indian mobile number.
    """
    clean = re.sub(r"[\s\-\(\)\.]", "", number)
    # Strip international prefix
    if clean.startswith("+91"):
        clean = clean[3:]
    elif clean.startswith("0091"):
        clean = clean[4:]
    elif clean.startswith("91") and len(clean) == 12:
        clean = clean[2:]
    elif clean.startswith("0") and len(clean) == 11:
        clean = clean[1:]

    # Must be exactly 10 digits after normalisation
    if not re.match(r"^[0-9]{10}$", clean):
        return False

    # First digit must be 6, 7, 8, or 9
    if clean[0] not in _MOBILE_VALID_FIRST_DIGITS:
        return False

    # Reject obviously invalid sequences (all same digit)
    if len(set(clean)) == 1:
        return False

    # Reject known test/dummy sequences
    dummy_sequences = {"1234567890", "0123456789", "0000000000"}
    if clean in dummy_sequences:
        return False

    return True


# ─────────────────────────────────────────────────────────────────────────────
# PINCODE (Indian Postal Code)
# ─────────────────────────────────────────────────────────────────────────────

def pincode_validate(pin: str) -> bool:
    """
    Validate Indian PIN code.

    Rules:
    - Exactly 6 digits
    - First digit must be 1-8 (no leading zero; 9xx not assigned)
    - Range: 110001 (Delhi) to 855117 (Assam/Bihar border)
    - 000000 and 999999 are never valid

    Args:
        pin: PIN code string (spaces stripped).

    Returns:
        True if plausible Indian PIN code.
    """
    clean = re.sub(r"[\s\-]", "", pin)
    if not re.match(r"^[0-9]{6}$", clean):
        return False
    pin_int = int(clean)
    # Valid range: 110001 to 855117
    if pin_int < 110001 or pin_int > 855117:
        return False
    return True


# ─────────────────────────────────────────────────────────────────────────────
# MICR CODE
# ─────────────────────────────────────────────────────────────────────────────

def micr_validate(micr: str) -> bool:
    """
    Validate Indian MICR code (Magnetic Ink Character Recognition).

    Format: 9 digits arranged as:
    - Digits 1-3: City code (001-999)
    - Digits 4-6: Bank code
    - Digits 7-9: Branch code

    Args:
        micr: MICR string (spaces stripped).

    Returns:
        True if exactly 9 digits with non-zero city code.
    """
    clean = re.sub(r"[\s]", "", micr)
    if not re.match(r"^[0-9]{9}$", clean):
        return False
    # City code (first 3) must not be 000
    if clean[:3] == "000":
        return False
    return True


# ─────────────────────────────────────────────────────────────────────────────
# SWIFT/BIC
# ─────────────────────────────────────────────────────────────────────────────

def swift_bic_validate(bic: str) -> bool:
    """
    Validate SWIFT/BIC code format.

    Format: AAAABBCC[DDD] (8 or 11 characters)
    - AAAA: Bank code (4 alpha)
    - BB: Country code (2 alpha, ISO 3166-1)
    - CC: Location code (2 alphanumeric)
    - DDD: Branch code (3 alphanumeric, optional; 'XXX' = primary)

    Args:
        bic: BIC string (case-insensitive, spaces stripped).

    Returns:
        True if valid BIC format.
    """
    clean = re.sub(r"[\s]", "", bic).upper()
    if len(clean) not in (8, 11):
        return False
    return bool(re.match(r"^[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?$", clean))


# ─────────────────────────────────────────────────────────────────────────────
# EMAIL
# ─────────────────────────────────────────────────────────────────────────────

# RFC 5322 simplified — captures all valid emails without catastrophic backtracking
_EMAIL_PATTERN = re.compile(
    r"^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]{1,64}"
    r"@"
    r"[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?"
    r"(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*"
    r"\.[a-zA-Z]{2,}$"
)


def email_validate(email: str) -> bool:
    """
    Validate email address format (RFC 5322 simplified).

    Rejects:
    - Missing @ symbol
    - Local part > 64 characters
    - Domain without TLD
    - Consecutive dots in domain

    Args:
        email: Email address string.

    Returns:
        True if syntactically valid email.
    """
    clean = email.strip()
    if len(clean) > 254:
        return False
    if ".." in clean:
        return False
    return bool(_EMAIL_PATTERN.match(clean))


# ─────────────────────────────────────────────────────────────────────────────
# CTRI (Clinical Trial Registration)
# ─────────────────────────────────────────────────────────────────────────────

def ctri_validate(ctri: str) -> bool:
    """
    Validate Clinical Trial Registry of India (CTRI) number.

    Format: CTRI/YYYY/MM/NNNNNN
    - YYYY: 4-digit year (2005 onwards — CTRI established 2007)
    - MM: 2-digit month (01-12)
    - NNNNNN: 6-digit sequential number

    Args:
        ctri: CTRI registration number string.

    Returns:
        True if valid CTRI format.
    """
    clean = ctri.strip().upper()
    m = re.match(r"^CTRI/(\d{4})/(\d{2})/(\d{6})$", clean)
    if not m:
        return False
    year, month = int(m.group(1)), int(m.group(2))
    return 2005 <= year <= 2030 and 1 <= month <= 12


# ─────────────────────────────────────────────────────────────────────────────
# AADHAAR AGE PLAUSIBILITY (cross-field)
# ─────────────────────────────────────────────────────────────────────────────

def aadhaar_age_plausibility(dob_str: str) -> bool:
    """
    Cross-field validator: checks if a DOB string implies a plausible age
    for an Aadhaar holder (0–120 years).

    Args:
        dob_str: Date of birth in DD/MM/YYYY, YYYY-MM-DD, or DD-MM-YYYY format.

    Returns:
        True if implied age is between 0 and 120 years.
    """
    import datetime
    clean = dob_str.strip()
    today = datetime.date.today()
    dob: Optional[datetime.date] = None
    for fmt in ("%d/%m/%Y", "%Y-%m-%d", "%d-%m-%Y", "%Y/%m/%d"):
        try:
            dob = datetime.datetime.strptime(clean, fmt).date()
            break
        except ValueError:
            continue
    if dob is None:
        return False
    age_years = (today - dob).days / 365.25
    return 0 <= age_years <= 120


# ─────────────────────────────────────────────────────────────────────────────
# BACKTRACKING SAFETY TEST
# ─────────────────────────────────────────────────────────────────────────────

def backtracking_safe(pattern_str: str, timeout_ms: int = 100) -> bool:
    """
    Test a compiled regex for catastrophic backtracking.

    Runs the pattern against a 20,000-character string of repeated 'a'
    characters. If the match takes longer than timeout_ms milliseconds,
    the pattern is considered unsafe.

    Args:
        pattern_str: Regex pattern string (not yet compiled).
        timeout_ms:  Maximum allowed match time in milliseconds.

    Returns:
        True if pattern is backtracking-safe.
    """
    import re as _re
    import time
    import signal
    import sys

    test_string = "a" * 20_000

    # Use timeout via signal on Unix, or time measurement on Windows
    start = time.monotonic()
    try:
        compiled = _re.compile(pattern_str)
        compiled.search(test_string)
        elapsed_ms = (time.monotonic() - start) * 1000
        return elapsed_ms < timeout_ms
    except Exception:
        return False


# ─────────────────────────────────────────────────────────────────────────────
# PUBLIC EXPORTS
# ─────────────────────────────────────────────────────────────────────────────

__all__ = [
    "verhoeff_validate",
    "verhoeff_generate_check_digit",
    "luhn_validate",
    "pan_checksum_validate",
    "gstin_validate",
    "ifsc_validate",
    "mobile_india_validate",
    "pincode_validate",
    "micr_validate",
    "swift_bic_validate",
    "email_validate",
    "ctri_validate",
    "aadhaar_age_plausibility",
    "backtracking_safe",
]
