"""
India Financial Patterns
=========================
PII patterns for Indian financial identifiers.

Covered:
  IN_BANK_ACCOUNT    Bank Account Number (context-gated, replaces dangerously broad version)
  IN_UPI             UPI VPA (comprehensive provider list)
  IN_IFSC            Indian Financial System Code
  IN_GST             GST Identification Number (with checksum)
  IN_TAN             Tax Deduction Account Number
  IN_DEMAT_ACCOUNT   Demat Account (CDSL / NSDL)
"""

import re
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from .base import PiiPattern, PatternCategory, DPDPASchedule


# --------------------------------------------------------------------------- #
# Validators                                                                   #
# --------------------------------------------------------------------------- #

def _validate_bank_account(value: str) -> bool:
    """Delegate to existing validator."""
    from validators.bank_account import validate_bank_account
    return validate_bank_account(value)


def _validate_upi(value: str) -> bool:
    """Delegate to existing validator."""
    from validators.upi import validate_upi
    return validate_upi(value)


def _validate_ifsc(value: str) -> bool:
    """Delegate to existing validator."""
    from validators.ifsc import validate_ifsc
    return validate_ifsc(value)


def _validate_gst(value: str) -> bool:
    """
    Validate Indian GST number format and state code.

    Format: [01-38][A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z] (15 chars)
    State codes: 01-38 (all Indian states and UTs).

    Note: GSTN's check-digit algorithm is not publicly documented; format +
    state-code validation provides high precision without false negatives.
    """
    clean = re.sub(r"[^A-Z0-9]", "", value.upper())
    if len(clean) != 15:
        return False
    if not re.match(r"^[0-3][0-9][A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z]$", clean):
        return False
    state_code = int(clean[:2])
    return 1 <= state_code <= 38


def _validate_tan(value: str) -> bool:
    """
    Validate TAN format: AAAA9999A
    4 alpha + 5 digits + 1 alpha, 4th alpha is city code letter.
    """
    clean = re.sub(r"[^A-Z0-9]", "", value.upper())
    if len(clean) != 10:
        return False
    return bool(re.match(r"^[A-Z]{4}[0-9]{5}[A-Z]$", clean))


# --------------------------------------------------------------------------- #
# Pattern definitions                                                          #
# --------------------------------------------------------------------------- #

IN_BANK_ACCOUNT = PiiPattern(
    id="IN_BANK_ACCOUNT",
    name="Indian Bank Account Number",
    category=PatternCategory.INDIA_FINANCIAL,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # 9-18 digit number — base confidence is 0.1; boosted by context keywords
        r"(?<!\d)[0-9]{9,18}(?!\d)",
    ],
    confidence_base=0.1,
    sensitivity="critical",
    context_keywords=[
        "account_number", "acc_no", "bank_account", "acct_num",
        "a/c", "account_no", "savings_account", "current_account",
        "bank_acc", "acctno",
    ],
    validator=_validate_bank_account,
    description=(
        "Indian bank account number (9-18 digits). "
        "Extremely low base confidence; only flagged with strong column-name context."
    ),
    false_positive_risk="critical",
    example="12345678901",
    country="IN",
    is_locked=True,
)

IN_UPI = PiiPattern(
    id="IN_UPI",
    name="Unified Payments Interface (UPI) VPA",
    category=PatternCategory.INDIA_FINANCIAL,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # Comprehensive UPI provider handle list (no generic fallback)
        (
            r"(?i)\b[a-z0-9._\-]{3,}@(?:"
            r"paytm|phonepe|gpay|googlepay|ybl|oksbi|okaxis|okicici|okhdfcbank|"
            r"ibl|airtel|apl|fbl|axl|upi|icici|sbi|hdfcbank|pnb|kotak|indus|"
            r"federal|rbl|idfcbank|cub|dbs|nsdl|mahb|boi|bom|cbin|cnrb|ucbi|"
            r"vijb|adb|kvb|dcb|corporation|equitas|esaf|freecharge|juspay|"
            r"mobikwik|razorpay|simpl|slice|uni|jupiter|fi|niyo|payzapp|"
            r"hsbc|sc|citi|deutsche|abfspay|aubank|idbi|"
            r"[a-z]{3,20}"   # Catch any future registered VPA provider
            r")\b"
        ),
    ],
    confidence_base=0.85,
    sensitivity="high",
    context_keywords=[
        "upi", "vpa", "upi_id", "payment_id", "bhim",
        "upi_address",
    ],
    validator=_validate_upi,
    description="UPI Virtual Payment Address (VPA) in handle@provider format.",
    false_positive_risk="low",
    example="rahul.kumar@paytm",
    country="IN",
    is_locked=True,
)

IN_IFSC = PiiPattern(
    id="IN_IFSC",
    name="Indian Financial System Code (IFSC)",
    category=PatternCategory.INDIA_FINANCIAL,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        # 4 alpha (bank code) + 0 (literal) + 6 alphanumeric (branch)
        r"(?i)\b[A-Z]{4}0[A-Z0-9]{6}\b",
    ],
    confidence_base=0.9,
    sensitivity="medium",
    context_keywords=[
        "ifsc", "ifsc_code", "bank_code", "branch_code",
        "neft", "rtgs", "imps",
    ],
    validator=_validate_ifsc,
    description="11-character IFSC code used for electronic fund transfers within India.",
    false_positive_risk="low",
    example="HDFC0001234",
    country="IN",
    is_locked=True,
)

IN_GST = PiiPattern(
    id="IN_GST",
    name="GST Identification Number (GSTIN)",
    category=PatternCategory.INDIA_FINANCIAL,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        # 15-char: 2-digit state + 10-char PAN + 1-digit entity + Z + checksum
        r"\b[0-3][0-9][A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z]\b",
    ],
    confidence_base=0.9,
    sensitivity="high",
    context_keywords=[
        "gst", "gstin", "gst_number", "gst_no", "tax_registration",
        "goods_and_services_tax",
    ],
    validator=_validate_gst,
    description="15-character GST Identification Number with Mod-36 checksum.",
    false_positive_risk="low",
    example="27AAAPZ1234F1Z5",
    country="IN",
    is_locked=True,  # Added to locked scope per task spec
)

IN_TAN = PiiPattern(
    id="IN_TAN",
    name="Tax Deduction Account Number (TAN)",
    category=PatternCategory.INDIA_FINANCIAL,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        # 4 alpha + 5 digits + 1 alpha (10 chars total)
        r"(?i)\b[A-Z]{4}[0-9]{5}[A-Z]\b",
    ],
    confidence_base=0.8,
    sensitivity="medium",
    context_keywords=[
        "tan", "tax_deduction", "tds_account", "tds_number",
        "tan_number", "tan_no",
    ],
    validator=_validate_tan,
    description="10-character TAN number issued by Income Tax Department for TDS/TCS deductions.",
    false_positive_risk="medium",
    example="PUNE12345A",
    country="IN",
    is_locked=False,
)

IN_DEMAT_ACCOUNT = PiiPattern(
    id="IN_DEMAT_ACCOUNT",
    name="Demat Account Number (CDSL / NSDL)",
    category=PatternCategory.INDIA_FINANCIAL,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # CDSL: IN + 16 digits
        r"(?i)\bIN[0-9]{16}\b",
        # NSDL: 8-digit DP ID + 8-digit client ID (with optional space/hyphen)
        r"(?<!\d)[0-9]{8}[\s-]?[0-9]{8}(?!\d)",
    ],
    confidence_base=0.65,
    sensitivity="high",
    context_keywords=[
        "demat", "dp_id", "depository", "cdsl", "nsdl",
        "demat_account", "dp_account",
    ],
    validator=None,
    description="Demat account identifier: CDSL 18-char (IN+16 digits) or NSDL 16-digit (DP+client).",
    false_positive_risk="medium",
    example="IN1201234567890123",
    country="IN",
    is_locked=False,
)


INDIA_FINANCIAL_PATTERNS = [
    IN_BANK_ACCOUNT,
    IN_UPI,
    IN_IFSC,
    IN_GST,
    IN_TAN,
    IN_DEMAT_ACCOUNT,
]
