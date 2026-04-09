"""
Global Financial Patterns
==========================
International financial identifiers: IBAN, SWIFT/BIC, US ABA routing,
and cryptocurrency wallet addresses.

Covered:
  IBAN              International Bank Account Number (ISO 13616, MOD-97)
  SWIFT_BIC         SWIFT Bank Identifier Code
  ABA_ROUTING       US ABA routing number (weighted mod-10)
  CRYPTO_BTC        Bitcoin address (P2PKH, P2SH, Bech32)
  CRYPTO_ETH        Ethereum address (EIP-55)
"""

import re
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from patterns.base import PiiPattern, PatternCategory, DPDPASchedule


# --------------------------------------------------------------------------- #
# Validators                                                                   #
# --------------------------------------------------------------------------- #

def _validate_iban(value: str) -> bool:
    """
    IBAN MOD-97 validation (ISO 13616).

    Algorithm:
    1. Remove spaces, convert to uppercase.
    2. Move first 4 chars to end.
    3. Convert letters to numbers (A=10, B=11, ... Z=35).
    4. Compute modulo 97; result must equal 1.

    Known valid: GB29NWBK60161331926819
    Known valid: DE89370400440532013000
    """
    clean = re.sub(r"\s", "", value.upper())
    if len(clean) < 15 or len(clean) > 34:
        return False
    if not re.match(r"^[A-Z]{2}[0-9]{2}[A-Z0-9]+$", clean):
        return False
    # Rearrange: move first 4 to end
    rearranged = clean[4:] + clean[:4]
    # Convert letters to digits
    numeric = ""
    for ch in rearranged:
        if ch.isdigit():
            numeric += ch
        else:
            numeric += str(ord(ch) - ord('A') + 10)
    return int(numeric) % 97 == 1


def _validate_swift_bic(value: str) -> bool:
    """
    SWIFT BIC format validation.
    Format: 4-letter bank code + 2-letter country + 2-char location + optional 3-char branch.
    8 or 11 characters total.
    """
    clean = re.sub(r"\s", "", value.upper())
    return bool(re.match(r"^[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?$", clean))


def _validate_aba_routing(value: str) -> bool:
    """
    US ABA routing number: 9 digits, weighted mod-10.
    Weights: 3, 7, 1 repeating (positions 0-8).
    Sum must be divisible by 10.

    Known valid: 021000021 (JPMorgan Chase NY)
    Known valid: 322271627 (Chase CA)
    """
    clean = re.sub(r"[\s-]", "", value)
    if not re.match(r"^[01][0-9]{8}$", clean):
        return False
    weights = [3, 7, 1, 3, 7, 1, 3, 7, 1]
    total = sum(int(clean[i]) * weights[i] for i in range(9))
    return total % 10 == 0


def _validate_btc_address(value: str) -> bool:
    """
    Basic Bitcoin address format validation.
    Accepts P2PKH (1...), P2SH (3...), and Bech32 (bc1...) formats.
    Does not perform elliptic-curve or checksum validation (no external deps).
    """
    clean = value.strip()
    # P2PKH: starts with 1, 25-34 chars
    if re.match(r"^1[a-km-zA-HJ-NP-Z1-9]{24,33}$", clean):
        return True
    # P2SH: starts with 3, 34 chars
    if re.match(r"^3[a-km-zA-HJ-NP-Z1-9]{33}$", clean):
        return True
    # Bech32: bc1q... or bc1p..., 14-74 chars after bc1
    if re.match(r"^bc1[ac-hj-np-z02-9]{6,87}$", clean.lower()):
        return True
    return False


def _validate_eth_address(value: str) -> bool:
    """
    Ethereum address: 0x followed by exactly 40 hex characters.
    """
    clean = value.strip()
    return bool(re.match(r"^0x[a-fA-F0-9]{40}$", clean))


# --------------------------------------------------------------------------- #
# Pattern definitions                                                          #
# --------------------------------------------------------------------------- #

IBAN = PiiPattern(
    id="IBAN",
    name="International Bank Account Number (IBAN)",
    category=PatternCategory.FINANCIAL_GLOBAL,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # Country code (2 alpha) + check digits (2 num) + BBAN (up to 30 alphanum)
        r"\b[A-Z]{2}[0-9]{2}[A-Z0-9]{1,30}\b",
    ],
    confidence_base=0.95,
    sensitivity="critical",
    context_keywords=[
        "iban", "bank_account", "account_number", "international_account",
        "iban_number",
    ],
    validator=_validate_iban,
    description="International Bank Account Number (ISO 13616) with MOD-97 validation.",
    false_positive_risk="low",
    example="GB29NWBK60161331926819",
    country="GLOBAL",
    is_locked=False,
)

SWIFT_BIC = PiiPattern(
    id="SWIFT_BIC",
    name="SWIFT / BIC Code",
    category=PatternCategory.FINANCIAL_GLOBAL,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        # 8 or 11 character BIC
        r"(?i)\b[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}(?:[A-Z0-9]{3})?\b",
    ],
    confidence_base=0.8,
    sensitivity="medium",
    context_keywords=[
        "swift", "bic", "bank_code", "correspondent_bank",
        "swift_code", "bic_code",
    ],
    validator=_validate_swift_bic,
    description="SWIFT Bank Identifier Code (8 or 11 chars) for international wire transfers.",
    false_positive_risk="medium",
    example="NWBKGB2L",
    country="GLOBAL",
    is_locked=False,
)

ABA_ROUTING = PiiPattern(
    id="ABA_ROUTING",
    name="US ABA Bank Routing Number",
    category=PatternCategory.FINANCIAL_GLOBAL,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        # 9 digits, starts with 0 or 1 (Federal Reserve district)
        r"\b[01][0-9]{8}\b",
    ],
    confidence_base=0.85,
    sensitivity="high",
    context_keywords=[
        "routing_number", "aba", "routing", "bank_routing",
        "aba_routing", "transit_number",
    ],
    validator=_validate_aba_routing,
    description="9-digit US ABA routing transit number with weighted mod-10 checksum.",
    false_positive_risk="medium",
    example="021000021",
    country="US",
    is_locked=False,
)

CRYPTO_BTC = PiiPattern(
    id="CRYPTO_BTC",
    name="Bitcoin Wallet Address",
    category=PatternCategory.FINANCIAL_GLOBAL,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # P2PKH: starts with 1
        r"\b1[a-km-zA-HJ-NP-Z1-9]{24,33}\b",
        # P2SH: starts with 3
        r"\b3[a-km-zA-HJ-NP-Z1-9]{33}\b",
        # Bech32: bc1...
        r"\bbc1[ac-hj-np-z02-9]{6,87}\b",
    ],
    confidence_base=0.9,
    sensitivity="high",
    context_keywords=[
        "bitcoin", "btc", "wallet", "crypto_address",
        "btc_address", "satoshi",
    ],
    validator=_validate_btc_address,
    description="Bitcoin wallet address in P2PKH (1...), P2SH (3...), or Bech32 (bc1...) format.",
    false_positive_risk="low",
    example="1A1zP1eP5QGefi2DMPTfTL5SLmv7Divf Na",
    country="GLOBAL",
    is_locked=False,
)

CRYPTO_ETH = PiiPattern(
    id="CRYPTO_ETH",
    name="Ethereum Wallet Address",
    category=PatternCategory.FINANCIAL_GLOBAL,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # 0x + 40 hex characters
        r"\b0x[a-fA-F0-9]{40}\b",
    ],
    confidence_base=0.9,
    sensitivity="high",
    context_keywords=[
        "ethereum", "eth", "wallet", "crypto_address",
        "eth_address", "ether",
    ],
    validator=_validate_eth_address,
    description="Ethereum wallet address: 0x prefix followed by 40 hex characters.",
    false_positive_risk="low",
    example="0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe",
    country="GLOBAL",
    is_locked=False,
)


FINANCIAL_GLOBAL_PATTERNS = [
    IBAN,
    SWIFT_BIC,
    ABA_ROUTING,
    CRYPTO_BTC,
    CRYPTO_ETH,
]
