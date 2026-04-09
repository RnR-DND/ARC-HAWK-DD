"""
IBAN Validator — MOD-97 (ISO 13616)
=====================================
Validates International Bank Account Numbers using the MOD-97-10 algorithm.

Reference: ISO 13616-1:2007
"""

import re


# ISO 3166-1 alpha-2 country codes with expected IBAN lengths
_IBAN_LENGTHS = {
    "AD": 24, "AE": 23, "AL": 28, "AT": 20, "AZ": 28,
    "BA": 20, "BE": 16, "BG": 22, "BH": 22, "BR": 29,
    "BY": 28, "CH": 21, "CR": 22, "CY": 28, "CZ": 24,
    "DE": 22, "DJ": 27, "DK": 18, "DO": 28, "DZ": 24,
    "EE": 20, "EG": 29, "ES": 24, "FI": 18, "FK": 18,
    "FO": 18, "FR": 27, "GB": 22, "GE": 22, "GI": 23,
    "GL": 18, "GR": 27, "GT": 28, "HR": 21, "HU": 28,
    "IE": 22, "IL": 23, "IQ": 23, "IS": 26, "IT": 27,
    "JO": 30, "KW": 30, "KZ": 20, "LB": 28, "LC": 32,
    "LI": 21, "LT": 20, "LU": 20, "LV": 21, "LY": 25,
    "MC": 27, "MD": 24, "ME": 22, "MK": 19, "MR": 27,
    "MT": 31, "MU": 30, "NI": 32, "NL": 18, "NO": 15,
    "OM": 23, "PK": 24, "PL": 28, "PS": 29, "PT": 25,
    "QA": 29, "RO": 24, "RS": 22, "RU": 33, "SA": 24,
    "SC": 31, "SD": 18, "SE": 24, "SI": 19, "SK": 24,
    "SM": 27, "SO": 23, "ST": 25, "SV": 28, "TL": 23,
    "TN": 24, "TR": 26, "UA": 29, "VA": 22, "VG": 24,
    "XK": 20,
}


def validate_iban(value: str) -> bool:
    """
    Validate an IBAN using the MOD-97-10 algorithm.

    Steps:
    1. Remove spaces, uppercase.
    2. Verify country code is known and length matches.
    3. Move first 4 chars to end.
    4. Convert letters to digits (A=10 … Z=35).
    5. Compute mod 97; result must be 1.

    Args:
        value: IBAN string, may contain spaces.

    Returns:
        True if structurally valid per ISO 13616.
    """
    clean = re.sub(r"\s+", "", value).upper()

    # Basic structure check
    if not re.match(r"^[A-Z]{2}[0-9]{2}[A-Z0-9]{11,30}$", clean):
        return False

    country = clean[:2]
    expected_len = _IBAN_LENGTHS.get(country)
    if expected_len is None:
        return False
    if len(clean) != expected_len:
        return False

    # Rearrange: move first 4 chars to end
    rearranged = clean[4:] + clean[:4]

    # Convert letters to digits
    numeric = ""
    for ch in rearranged:
        if ch.isalpha():
            numeric += str(ord(ch) - ord("A") + 10)
        else:
            numeric += ch

    # MOD-97 check
    return int(numeric) % 97 == 1
