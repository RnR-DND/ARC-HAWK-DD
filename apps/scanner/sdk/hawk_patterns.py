"""
Hawk PII Pattern Library — Single Source of Truth
===================================================
THE authoritative file for all PII pattern definitions in the Hawk platform.

No other file should define PII patterns. All pattern references across the
scanner pipeline must import from this module.

Architecture:
  hawk_patterns.py    ← you are here (30+ PatternDef instances)
  hawk_validators.py  ← mathematical validators (Verhoeff, Luhn, etc.)
  hawk_patterns_test.py ← startup self-test runner

Usage:
  from sdk.hawk_patterns import ALL_PATTERNS, get_pattern, get_by_category
  from sdk.hawk_patterns import PatternDef, DPDPASchedule

Pattern fields:
  id                  Unique uppercase identifier
  name                Human-readable display name
  category            Pattern category string
  dpdpa_schedule      DPDPA 2023 schedule classification
  patterns            List of RE2-safe regex strings
  flags               Regex flags list (e.g. ["IGNORECASE"])
  confidence_regex    Confidence when regex matches but no validator
  confidence_validated Confidence when regex + validator both pass
  sensitivity         "critical" | "high" | "medium" | "low"
  context_keywords    Column/table name substrings that boost confidence
  validator           Callable(str) -> bool or None
  pii_category        DPDPA-aligned category name
  description         One-sentence description
  false_positive_risk "low" | "medium" | "high" | "critical"
  example             A syntactically valid example value
  country             ISO-3166-1 alpha-2 or "GLOBAL"
  is_locked           If True, cannot be disabled by user configuration
  test_positives      ≥10 real-format samples that MUST match
  test_negatives      ≥10 known false-positive-prone strings that must NOT match
  notes               Why this pattern exists, edge cases, design decisions

Performance targets:
  Field mode:   ≥ 10,000 field values/second on patterns alone
  Context mode: ≥ 1,000 document chunks/second
  All patterns pass backtracking safety test (20k char 'a' string < 100ms)
"""

from __future__ import annotations

import re
import sys
import os
from dataclasses import dataclass, field
from typing import Callable, Dict, List, Optional

# Add sdk to path for validator imports
_SDK_DIR = os.path.dirname(os.path.abspath(__file__))
if _SDK_DIR not in sys.path:
    sys.path.insert(0, _SDK_DIR)

# Import all validators
from hawk_validators import (
    verhoeff_validate,
    luhn_validate,
    pan_checksum_validate,
    gstin_validate,
    ifsc_validate,
    mobile_india_validate,
    pincode_validate,
    micr_validate,
    swift_bic_validate,
    email_validate,
    ctri_validate,
)


# ─────────────────────────────────────────────────────────────────────────────
# CORE DATA TYPES
# ─────────────────────────────────────────────────────────────────────────────

class DPDPASchedule:
    PERSONAL_DATA = "Personal Data"
    SENSITIVE_PERSONAL_DATA = "Sensitive Personal Data"
    CRITICAL_PERSONAL_DATA = "Critical Personal Data"
    CHILDRENS_DATA = "Children's Data"


@dataclass
class PatternDef:
    """
    Complete descriptor for a single PII pattern in the Hawk library.

    Every high-sensitivity pattern (confidence_validated ≥ 0.90) MUST have
    a validator. Patterns without a validator are capped at confidence 0.75
    for custom patterns and 0.85 for built-in patterns.
    """
    # Identity
    id: str
    name: str
    category: str                    # Human-readable category name
    pii_category: str                # DPDPA-aligned PII category label
    dpdpa_schedule: str              # One of DPDPASchedule constants
    sensitivity: str                 # "critical" | "high" | "medium" | "low"

    # Pattern
    patterns: List[str]              # RE2-safe regex strings
    flags: List[str] = field(default_factory=list)  # ["IGNORECASE", "MULTILINE"]

    # Confidence scoring
    confidence_regex: float = 0.60   # Confidence: regex match only, no validator
    confidence_validated: float = 0.97  # Confidence: regex + validator pass

    # Detection aids
    context_keywords: List[str] = field(default_factory=list)
    validator: Optional[Callable] = None

    # Documentation
    description: str = ""
    false_positive_risk: str = "medium"
    example: str = ""
    country: str = "IN"
    is_locked: bool = False
    notes: str = ""

    # Self-testing (REQUIRED for all patterns)
    test_positives: List[str] = field(default_factory=list)  # ≥10 MUST match
    test_negatives: List[str] = field(default_factory=list)  # ≥10 MUST NOT match

    def compile(self) -> List[re.Pattern]:
        """Return compiled re.Pattern objects."""
        compiled_flags = 0
        if "IGNORECASE" in self.flags:
            compiled_flags |= re.IGNORECASE
        if "MULTILINE" in self.flags:
            compiled_flags |= re.MULTILINE
        return [re.compile(p, compiled_flags) for p in self.patterns]

    def validate(self, value: str) -> bool:
        """Run validator if present. Returns True if valid or no validator."""
        if self.validator is None:
            return True
        try:
            return bool(self.validator(value))
        except Exception:
            return False

    def get_confidence(self, validated: bool) -> float:
        """Return appropriate confidence based on whether validator passed."""
        if validated and self.validator is not None:
            return self.confidence_validated
        return self.confidence_regex


# ─────────────────────────────────────────────────────────────────────────────
# HELPER: VALIDATOR WRAPPERS
# ─────────────────────────────────────────────────────────────────────────────

def _aadhaar_validate(value: str) -> bool:
    """Aadhaar: strip spaces/hyphens, check 12 digits, Verhoeff checksum."""
    clean = re.sub(r"[\s\-]", "", value)
    if len(clean) != 12 or clean[0] in ("0", "1"):
        return False
    return verhoeff_validate(clean)


def _always_valid(_: str) -> bool:
    """For patterns with near-zero FP rate due to unique structural prefix."""
    return True


def _pan_validate(value: str) -> bool:
    clean = re.sub(r"[\s]", "", value).upper()
    return pan_checksum_validate(clean)


def _gstin_validate(value: str) -> bool:
    return gstin_validate(value)


def _ifsc_validate(value: str) -> bool:
    return ifsc_validate(value)


def _luhn_validate(value: str) -> bool:
    clean = re.sub(r"[\s\-]", "", value)
    return luhn_validate(clean)


def _mobile_validate(value: str) -> bool:
    return mobile_india_validate(value)


def _pincode_validate(value: str) -> bool:
    return pincode_validate(value)


def _micr_validate(value: str) -> bool:
    return micr_validate(value)


def _swift_validate(value: str) -> bool:
    return swift_bic_validate(value)


def _email_validate(value: str) -> bool:
    return email_validate(value)


def _ctri_validate(value: str) -> bool:
    return ctri_validate(value)


def _dob_validate(value: str) -> bool:
    import datetime
    clean = value.strip()
    for fmt in ("%d/%m/%Y", "%Y-%m-%d", "%d-%m-%Y", "%d.%m.%Y"):
        try:
            dt = datetime.datetime.strptime(clean, fmt).date()
            age = (datetime.date.today() - dt).days / 365.25
            return 0 <= age <= 120
        except ValueError:
            continue
    return False


def _passport_validate(value: str) -> bool:
    clean = re.sub(r"[^A-Za-z0-9]", "", value).upper()
    if len(clean) != 9:
        return False
    if clean[0] in ('Q', 'X', 'Z'):
        return False
    return bool(re.match(r"^[A-PR-WY][1-9][0-9]{5}[1-9][A-Z]$", clean))


def _cin_validate(value: str) -> bool:
    clean = re.sub(r"\s", "", value).upper()
    if len(clean) != 21:
        return False
    if not re.match(r"^[LU][0-9]{5}[A-Z]{2}[0-9]{4}[A-Z]{3}[0-9]{6}$", clean):
        return False
    valid_types = {"PLC", "PTC", "LTD", "LLC", "OPC", "NPL", "FLC", "FTC", "GOI"}
    return clean[15:18] in valid_types


# ─────────────────────────────────────────────────────────────────────────────
# SECTION 1: INDIA GOVERNMENT IDENTITY DOCUMENTS
# ─────────────────────────────────────────────────────────────────────────────

IN_AADHAAR = PatternDef(
    id="IN_AADHAAR",
    name="Aadhaar Number",
    category="India Identity",
    pii_category="Unique Identifier",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="critical",
    patterns=[
        # 12-digit, first digit 2-9, allows single space/hyphen separators
        r"(?<!\d)[2-9]\d{3}[\s\-]?\d{4}[\s\-]?\d{4}(?!\d)",
    ],
    flags=[],
    confidence_regex=0.60,
    confidence_validated=0.98,
    context_keywords=[
        "aadhaar", "aadhar", "uid", "uidai",
        "unique identification", "unique identity", "enrollment", "enrolment",
    ],
    validator=_aadhaar_validate,
    description="12-digit UIDAI-issued unique identification number with Verhoeff checksum.",
    false_positive_risk="low",
    example="2345 6789 0126",
    country="IN",
    is_locked=True,
    notes=(
        "First digit must be 2-9 (UIDAI never issues 0xxx or 1xxx). "
        "Verhoeff checksum validation reduces FP rate dramatically vs RE2. "
        "Obfuscated forms (XXXX XXXX 1234) handled by context-window mode."
    ),
    test_positives=[
        "2345 6789 0126",
        "234567890126",
        "5234-5678-9015",
        "9876 5432 1095",
        "2000 1234 5677",
        "8765 4321 0987",
        "6543 2109 8765",
        "4321 0987 6543",
        "7890 1234 5670",
        "3456 7890 1234",
    ],
    test_negatives=[
        "0123 4567 8901",   # starts with 0 — regex rejects [2-9] first digit
        "1234 5678 9012",   # starts with 1 — regex rejects [2-9] first digit
        "INV-2024-000123",  # no 12-digit sequence
        "PIN: 110001",      # only 6 digits
        "ABCDE1234F",       # all alpha — not digits
        "2025-01-15",       # date, only 8 digits
        "Rs. 23456789",     # only 8 digits
        "12 digits: 012345678901",  # starts with 0
        "ver 1.2.3.4",      # not digits
        "phone: 0987654321",  # starts with 0 after 0-prefix
    ],
)

IN_PAN = PatternDef(
    id="IN_PAN",
    name="Permanent Account Number (PAN)",
    category="India Identity",
    pii_category="Tax Identifier",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="critical",
    patterns=[
        # 5 alpha (AAANN pattern) + 4 digits + 1 alpha, word boundary
        r"(?i)\b[A-Z]{5}[0-9]{4}[A-Z]\b",
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.70,
    confidence_validated=0.97,
    context_keywords=[
        "pan", "pancard", "pan card", "permanent account",
        "income tax", "tax id", "tax number",
    ],
    validator=_pan_validate,
    description="10-character alphanumeric tax identifier issued by India's Income Tax Department.",
    false_positive_risk="low",
    example="ABCDE1234F",
    country="IN",
    is_locked=True,
    notes=(
        "PAN entity code (4th char) must be in {A,B,C,F,G,H,J,L,P,T,K,E}. "
        "Invalid sequential numbers (0000) rejected. "
        "Pattern overlaps with some serial numbers — context validation required."
    ),
    test_positives=[
        "ABCDE1234F",
        "AAAPA1234A",
        "BBBPB9999B",
        "CCCPC1234C",
        "DDDPD5678D",
        "EEEHE3456E",
        "FFFGF7890F",
        "GGGCG1111G",
        "HHHFH2222H",
        "AAACM1234A",
    ],
    test_negatives=[
        "1234567890",    # all digits — no alpha component
        "ABCD12345",     # 9 chars — too short
        "ABCDE12345",    # 10 chars but ends in digit (not alpha final)
        "Invoice#ABC",   # has # and no digit block
        "27AAAPZ1234F1Z5",     # GSTIN (starts with digits — won't match \b[A-Z]{5})
        "ABCDE1234",     # only 9 chars
        "A1B2C3D4E5",    # mixed alternating — not 5 alpha + 4 digit + 1 alpha
        "12345ABCDE",    # digits first — won't match \b[A-Z]{5}
        "ABC123",        # too short
        "ABCDE123",      # only 8 chars
    ],
)

IN_PASSPORT = PatternDef(
    id="IN_PASSPORT",
    name="Indian Passport Number",
    category="India Identity",
    pii_category="Travel Document",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="critical",
    patterns=[
        # [A-PR-WY] first letter (excludes Q,X,Z) + digit + 5 digits + digit + alpha
        r"(?i)\b[A-PR-WYa-pr-wy][1-9][0-9]{5}[1-9][A-Za-z]\b",
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.65,
    confidence_validated=0.95,
    context_keywords=[
        "passport", "passport_no", "passport_number", "pp_no",
        "travel_document", "ppno",
    ],
    validator=_passport_validate,
    description="9-character Indian passport number issued by MEA. First letter excludes Q, X, Z.",
    false_positive_risk="medium",
    example="A1234567B",
    country="IN",
    is_locked=True,
    notes="Letters Q, X, Z are not issued in first position by MEA. First and 8th position must be non-zero digit.",
    test_positives=[
        "A1234567B",
        "B2345678C",
        "C3456789D",
        "P1234567A",
        "S9876543Z",
        "R5678901M",
        "W1234567N",
        "A9999991A",
        "M1111111B",
        "Y2222222C",
    ],
    test_negatives=[
        "Q1234567A",    # Q not issued
        "X1234567A",    # X not issued
        "Z1234567A",    # Z not issued
        "A0234567B",    # second char is 0
        "A1234567",     # only 8 chars
        "A12345678B",   # 10 chars
        "ABCDE1234F",   # PAN format
        "1234567890",   # all digits
        "AA1234567B",   # two leading alpha
        "Invoice#001",
    ],
)

IN_VOTER_ID = PatternDef(
    id="IN_VOTER_ID",
    name="Voter ID / EPIC",
    category="India Identity",
    pii_category="Electoral Document",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="high",
    patterns=[
        # 3 uppercase alpha + 7 digits — strict uppercase to reduce FP
        r"\b[A-Z]{3}[0-9]{7}\b",
    ],
    flags=[],
    confidence_regex=0.50,
    confidence_validated=0.80,
    context_keywords=[
        "voter_id", "voter id", "epic", "election", "voter_card",
        "electors", "ec_id",
    ],
    validator=None,
    description="Electors Photo Identity Card (EPIC) number issued by Election Commission of India.",
    false_positive_risk="high",
    example="ABC1234567",
    country="IN",
    is_locked=True,
    notes="High FP risk without context — AAA0000000 format matches many serial numbers. Context-gated.",
    test_positives=[
        "ABC1234567",
        "DEF2345678",
        "GHI3456789",
        "MNO1111111",
        "PQR2222222",
        "STU3333333",
        "VWX4444444",
        "YZA5555555",
        "XYZ6666666",
        "BCD7890123",
    ],
    test_negatives=[
        "abc1234567",   # lowercase (pattern requires uppercase)
        "AB1234567",    # only 2 alpha — needs exactly 3
        "ABCD123456",   # 4 alpha — too many alpha chars before digits
        "ABC123456",    # only 6 digits — needs exactly 7
        "ABC12345678",  # 8 digits — too many
        "1234567890",   # no alpha prefix
        "HDFC0001234",  # IFSC (4 alpha + 7 mixed chars, not 3+7)
        "ABCDE1234F",   # PAN (5 alpha + 4 digit + 1 alpha)
        "27AAAPZ1234",  # digits-first
        "AB-12-34-567", # separator format, not 3+7
    ],
)

IN_DRIVING_LICENSE = PatternDef(
    id="IN_DRIVING_LICENSE",
    name="Indian Driving Licence",
    category="India Identity",
    pii_category="Driving Document",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="high",
    patterns=[
        # Full format: 2-alpha state + 2-digit RTO + 4-digit year + 7-digit serial
        r"(?i)\b[A-Z]{2}[\s\-]?[0-9]{2}[\s\-]?[0-9]{4}[\s\-]?[0-9]{7}\b",
        # Compact (no separators, 15 digits after 2 alpha)
        r"(?i)\b[A-Z]{2}[0-9]{13}\b",
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.60,
    confidence_validated=0.85,
    context_keywords=[
        "driving_license", "driving_licence", "dl_no", "dl_number",
        "license_number", "licence_number", "dl",
    ],
    validator=None,
    description="Indian driving licence number: state-code (2) + RTO (2) + year (4) + serial (7).",
    false_positive_risk="medium",
    example="MH-12-2019-1234567",
    country="IN",
    is_locked=True,
    notes="Format varies slightly by state RTO. Year range 1990-2030 is realistic.",
    test_positives=[
        "MH-12-2019-1234567",
        "DL-01-2020-0012345",
        "KA-05-2018-9876543",
        "TN-09-2021-1111111",
        "GJ-06-2017-2222222",
        "UP-80-2022-3333333",
        "WB-08-2016-4444444",
        "RJ-14-2019-5555555",
        "MH1220191234567",    # compact: 2-alpha + 13 digits
        "DL0120200012345",    # compact: 2-alpha + 13 digits
    ],
    test_negatives=[
        "ABCDE1234F",        # PAN — alpha pattern
        "MH-12-19-123456",   # year is 2 digits not 4
        "MH-AB-2019-1234567",  # RTO is alpha not numeric
        "Phone: 9876543210",   # no DL format
        "2019-12-01",          # date, no alpha prefix
        "MH1",                 # too short
        "12-2019-1234567",     # missing state alpha prefix
        "M1-12-2019-1234567",  # only 1 alpha state code
        "MHXX2019XXXXXXX",     # alpha in RTO/serial positions
        "MH122019",            # too short compact (8 chars after MH)
    ],
)

IN_GSTIN = PatternDef(
    id="IN_GSTIN",
    name="GST Identification Number (GSTIN)",
    category="India Identity",
    pii_category="Tax Identifier",
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    sensitivity="high",
    patterns=[
        # 15-char: 2-digit state + 10-char PAN + 1-digit entity + Z + checksum
        r"\b[0-3][0-9][A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z]\b",
    ],
    flags=[],
    confidence_regex=0.85,
    confidence_validated=0.97,
    context_keywords=[
        "gst", "gstin", "gst_number", "gst_no", "tax_registration",
        "goods_and_services_tax",
    ],
    validator=_gstin_validate,
    description="15-character GST Identification Number with embedded PAN and state code.",
    false_positive_risk="low",
    example="27AAAPZ1234F1Z5",
    country="IN",
    is_locked=True,
    notes="State code must be 01-38 (all Indian states+UTs). Embeds full PAN in positions 3-12.",
    test_positives=[
        "27AAAPZ1234F1Z5",
        "29AAAPA1234A1Z6",
        "07BBBPB1234B1Z4",
        "33CCCPC5678C1Z8",
        "19DDDPD9012D1Z7",
        "24AAACM1234A1Z9",
        "09BBBPC9999B1Z3",
        "22EEEHE3456E1Z2",
        "06FFFGF7890F1Z1",
        "37GGGCG1111G1ZA",
    ],
    test_negatives=[
        "27AAAPZ1234F1A5",  # 'A' not 'Z' at position 14 — regex requires Z
        "ABCDE1234F",       # PAN (no leading digits, only 10 chars)
        "1234567890",       # only 10 digits, no embedded PAN
        "HDFC0001234",      # IFSC — 11 chars only
        "ABC1234567",       # voter ID — 10 chars only
        "27AAAPZ1234",      # too short (12 chars)
        "27AAAPZ1234F1Z",   # 14 chars missing checksum
        "GSTIN:27AAAPZ",    # incomplete
        "27-AAAPZ-1234F",   # hyphens not matched
        "27aaapz1234f1z5",  # lowercase — pattern requires uppercase
    ],
)

IN_CIN = PatternDef(
    id="IN_CIN",
    name="Company Identification Number (CIN)",
    category="India Corporate",
    pii_category="Corporate Identifier",
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    sensitivity="medium",
    patterns=[
        # L/U + 5-digit NIC + 2-alpha state + 4-digit year + 3-alpha type + 6 digits
        r"(?i)\b[LUlu][0-9]{5}[A-Z]{2}[0-9]{4}[A-Z]{3}[0-9]{6}\b",
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.85,
    confidence_validated=0.96,
    context_keywords=[
        "cin", "company_id", "mca", "corporate_identity",
        "company_identification", "cin_number",
    ],
    validator=_cin_validate,
    description="21-character CIN issued by MCA: L/U + NIC + state + year + company-type + serial.",
    false_positive_risk="low",
    example="L17110MH1973PLC019786",
    country="IN",
    is_locked=False,
    notes="Company type must be in {PLC, PTC, LTD, LLC, OPC, NPL, FLC, FTC, GOI}.",
    test_positives=[
        "L17110MH1973PLC019786",
        "U72900KA2010PTC054321",
        "L74140DL1995PLC069678",
        "U67190TN2005PTC054321",
        "L36911KA1985PLC006227",
        "U19114AP2007PLC056789",
        "L31300MH1983PLC031234",
        "U72200DL2001PLC123456",
        "L40100GJ1994PLC021234",
        "U72300MH2010OPC012345",
    ],
    test_negatives=[
        "A17110MH1973PLC019786",  # starts with A not L/U — regex requires [LU]
        "L17110MH73PLC019786",    # year only 2 digits (4 required)
        "ABCDE1234F",             # PAN — only 10 chars
        "1234567890123456789022", # 22 chars but all digits — no state code
        "L1711MH1973PLC019786",   # NIC only 4 digits (needs 5)
        "Invoice-L17110MH1973",   # partial, not CIN format
        "CIN:U72900",             # partial
        "27AAAPZ1234F1Z5",        # GSTIN — starts with digits
        "L17110MH1973PLCxyz786",  # lowercase in serial — pattern requires digits in serial
        "U72900KA20101TC054321",  # 'T' not valid [A-Z]{3} type pattern boundary
    ],
)

IN_IFSC = PatternDef(
    id="IN_IFSC",
    name="Indian Financial System Code (IFSC)",
    category="India Financial",
    pii_category="Bank Code",
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    sensitivity="medium",
    patterns=[
        # 4 alpha (bank) + 0 (literal) + 6 alphanumeric (branch)
        r"(?i)\b[A-Z]{4}0[A-Z0-9]{6}\b",
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.85,
    confidence_validated=0.97,
    context_keywords=[
        "ifsc", "ifsc_code", "bank_code", "branch_code",
        "neft", "rtgs", "imps",
    ],
    validator=_ifsc_validate,
    description="11-character IFSC code: 4-alpha bank + '0' + 6-alphanumeric branch.",
    false_positive_risk="low",
    example="HDFC0001234",
    country="IN",
    is_locked=True,
    notes="5th character is always '0' (zero). This uniquely identifies IFSC among similar patterns.",
    test_positives=[
        "HDFC0001234",
        "SBIN0012345",
        "ICIC0006789",
        "UTIB0001234",
        "KKBK0005678",
        "PUNB0123456",
        "CNRB0019012",
        "BARB0NANDUR",
        "CITI0000001",
        "AXIS0001234",
    ],
    test_negatives=[
        "HDFC1001234",  # 5th char is 1 not 0
        "HDFC0",        # too short
        "HDFC00012345",  # too long (12 chars)
        "1234567890A",   # starts with digit
        "ABCDE1234F",    # PAN (5 alpha + 4 digit + 1 alpha)
        "27AAAPZ1234F",  # GSTIN prefix
        "L17110MH1973",  # CIN prefix
        "SWIFT CODE: HDFC",
        "HDFC-0-001234",  # with hyphens (different separator)
        "HDFC_0001234",   # underscore
    ],
)

IN_UPI = PatternDef(
    id="IN_UPI",
    name="UPI Virtual Payment Address (VPA)",
    category="India Financial",
    pii_category="Payment Identifier",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="high",
    patterns=[
        # Comprehensive UPI provider handle list
        (
            r"(?i)\b[a-z0-9._\-]{3,}@(?:"
            r"paytm|phonepe|gpay|googlepay|ybl|oksbi|okaxis|okicici|okhdfcbank|"
            r"ibl|airtel|apl|fbl|axl|upi|icici|sbi|hdfcbank|pnb|kotak|indus|"
            r"federal|rbl|idfcbank|cub|dbs|nsdl|mahb|boi|bom|cbin|cnrb|ucbi|"
            r"vijb|adb|kvb|dcb|corporation|equitas|esaf|freecharge|juspay|"
            r"mobikwik|razorpay|simpl|slice|uni|jupiter|fi|niyo|payzapp|"
            r"hsbc|sc|citi|deutsche|abfspay|aubank|idbi"
            r")(?!\.[a-z])\b"
        ),
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.85,
    confidence_validated=0.95,
    context_keywords=[
        "upi", "vpa", "upi_id", "payment_id", "bhim",
        "upi_address",
    ],
    validator=None,
    description="UPI Virtual Payment Address (VPA) in handle@provider format.",
    false_positive_risk="low",
    example="rahul.kumar@paytm",
    country="IN",
    is_locked=True,
    notes="Provider list updated to include all NPCI-registered VPA handles as of 2024.",
    test_positives=[
        "rahul.kumar@paytm",
        "user@phonepe",
        "name@gpay",
        "abc123@ybl",
        "test@oksbi",
        "john.doe@okaxis",
        "priya_123@kotak",
        "business@razorpay",
        "user.name@idfcbank",
        "mobile@airtel",
    ],
    test_negatives=[
        "user@gmail.com",       # gmail has .com suffix — (?!\.[a-z]) rejects
        "user@outlook.com",     # .com suffix rejected
        "admin@company.co.in",  # .co TLD after provider rejected
        "test@test.com",        # .com suffix rejected
        "abc@12",               # provider starts with digit (not alpha)
        "user",                 # no @ sign
        "@paytm",               # no local part (< 3 chars before @)
        "a@paytm",              # local part only 1 char (< 3)
        "user@",                # no provider
        "contact@company.org",  # .org suffix rejected
    ],
)

IN_RATION_CARD = PatternDef(
    id="IN_RATION_CARD",
    name="Ration Card Number",
    category="India Identity",
    pii_category="Government Welfare Document",
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    sensitivity="medium",
    patterns=[
        # State prefix (2-3 alpha) + 8-10 digits, optional separator
        r"(?i)\b[A-Z]{2,3}[\s\-]?[0-9]{8,10}\b",
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.40,
    confidence_validated=0.70,
    context_keywords=[
        "ration_card", "ration_no", "ration card", "pds",
        "fair_price", "food_security",
    ],
    validator=None,
    description="State-issued ration card number: state prefix (2-3 alpha) + 8-10 digit sequence.",
    false_positive_risk="high",
    example="MH12345678",
    country="IN",
    is_locked=False,
    notes="Very broad pattern — only reliable with strong column-name context.",
    test_positives=[
        "MH12345678",
        "TN123456789",
        "KA1234567890",
        "DL12345678",
        "UP12345678",
        "WB123456789",
        "AP12345678",
        "GJ1234567890",
        "RJ12345678",
        "MP123456789",
    ],
    test_negatives=[
        "1234567890",          # no alpha prefix
        "ABCDEFGH",            # no digits
        "MH1234567",           # only 7 digits
        "ABCDE1234F",          # PAN
        "MH-12-2019-1234567",  # driving licence
        "ABC1234567",          # voter ID (3 alpha + 7 digits)
        "+91 9876543210",      # phone
        "MH1",                 # too short
        "HDFC0001234",         # IFSC
        "2345 6789 0126",      # Aadhaar
    ],
)

IN_UAN = PatternDef(
    id="IN_UAN",
    name="Universal Account Number (UAN) — EPFO",
    category="India Identity",
    pii_category="Employment Identifier",
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    sensitivity="medium",
    patterns=[
        # 12-digit number — very broad, must rely on context
        r"(?<!\d)[0-9]{12}(?!\d)",
    ],
    flags=[],
    confidence_regex=0.20,
    confidence_validated=0.70,
    context_keywords=[
        "uan", "epf", "provident_fund", "pf_account",
        "employee_provident", "epfo",
    ],
    validator=None,
    description="12-digit Universal Account Number issued by EPFO. Low base confidence; context-gated.",
    false_positive_risk="critical",
    example="100123456789",
    country="IN",
    is_locked=False,
    notes="Extremely broad pattern (any 12-digit number). Use only with EPFO/UAN column context.",
    test_positives=[
        "100123456789",
        "200987654321",
        "100000000001",
        "999999999999",
        "123456789012",
        "234567890123",
        "345678901234",
        "456789012345",
        "567890123456",
        "678901234567",
    ],
    test_negatives=[
        "1234567890",       # 10 digits
        "1234567890123",    # 13 digits
        "1234 5678 9012",   # would match as 12 digit with spaces stripped
        "ABCDE1234F",       # PAN
        "+91-9876543210",   # phone (starts with +91)
        "Rs. 12345678",     # amount
        "2025-01-15",       # date
        "Tax: 1234",
        "Invoice #123456",
        "123 456 789",      # only 9 digits
    ],
)

IN_ABHA = PatternDef(
    id="IN_ABHA",
    name="Ayushman Bharat Health Account (ABHA) ID",
    category="India Healthcare",
    pii_category="Health Identifier",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="critical",
    patterns=[
        # 14-digit, displayed as XX-XXXX-XXXX-XXXX
        r"(?<!\d)[0-9]{2}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}(?!\d)",
        # Compact 14 digits
        r"(?<!\d)[0-9]{14}(?!\d)",
    ],
    flags=[],
    confidence_regex=0.65,
    confidence_validated=0.90,
    context_keywords=[
        "abha", "health_id", "ayushman", "health_account",
        "ndhm", "health_id_number", "abdm",
    ],
    validator=None,
    description="14-digit Ayushman Bharat Health Account ID issued by NHA under ABDM.",
    false_positive_risk="medium",
    example="91-2345-6789-0123",
    country="IN",
    is_locked=False,
    notes="Displayed as 14 digits in XX-XXXX-XXXX-XXXX format. High value health data.",
    test_positives=[
        "91-2345-6789-0123",
        "12-3456-7890-1234",
        "91234567890123",
        "43-9876-5432-1098",
        "56-1234-5678-9012",
        "78-9012-3456-7890",
        "89-0123-4567-8901",
        "90-1234-5678-9012",
        "23-4567-8901-2345",
        "34-5678-9012-3456",
    ],
    test_negatives=[
        "1234567890123",    # 13 digits — too short
        "123456789012345",  # 15 digits — one too many
        "+91 98765 43210",  # phone — only 10 digits + prefix
        "2345 6789 0126",   # Aadhaar — 12 digits with spaces
        "ABCDE1234F",       # PAN — alpha chars
        "91.2345.6789",     # has dots — not 14 digits
        "INV-12345678901",  # 11 digits after prefix, not 14
        "ABHA-12345",       # too short
        "91 23 45 67",      # spaces, only 8 digits
        "1234567890",       # 10 digits only
    ],
)


# ─────────────────────────────────────────────────────────────────────────────
# SECTION 2: INDIA FINANCIAL DATA
# ─────────────────────────────────────────────────────────────────────────────

IN_CREDIT_CARD = PatternDef(
    id="IN_CREDIT_CARD",
    name="Credit Card Number",
    category="Financial Global",
    pii_category="Payment Card",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="critical",
    patterns=[
        # Visa: 4xxx, 13 or 16 digits
        r"\b4[0-9]{3}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}[\s\-]?[0-9]{1,4}\b",
        # Mastercard: 5[1-5]xx or 2[2-7]xx
        r"\b(?:5[1-5][0-9]{2}|2[2-7][0-9]{2})[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}\b",
        # Amex: 3[47]xx (15 digits)
        r"\b3[47][0-9]{2}[\s\-]?[0-9]{6}[\s\-]?[0-9]{5}\b",
        # Discover: 6011, 65xx, 64[4-9]x
        r"\b(?:6011|65[0-9]{2}|64[4-9][0-9])[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}\b",
        # RuPay (India): 60, 65, 81, 82, 508
        r"\b(?:60[0-9]{2}|65[0-9]{2}|8[12][0-9]{2}|508[0-9])[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}\b",
    ],
    flags=[],
    confidence_regex=0.60,
    confidence_validated=0.97,
    context_keywords=[
        "credit_card", "card_number", "card_no", "cc_number",
        "visa", "mastercard", "amex", "rupay", "discover",
        "payment_card",
    ],
    validator=_luhn_validate,
    description="Credit card number (Visa/MC/Amex/Discover/RuPay) with Luhn checksum validation.",
    false_positive_risk="medium",
    example="4532015112830366",
    country="GLOBAL",
    is_locked=True,
    notes="Luhn validation eliminates most false positives. RuPay (India's domestic card) included.",
    test_positives=[
        "4532015112830366",    # Visa (Luhn valid)
        "4916338506082832",    # Visa
        "5425233430109903",    # Mastercard (Luhn valid)
        "2223000048400011",    # Mastercard 2-series
        "371449635398431",     # Amex (Luhn valid)
        "6011111111111117",    # Discover (Luhn valid)
        "4532 0151 1283 0366", # With spaces
        "4532-0151-1283-0366", # With hyphens
        "4916338506082832",
        "5425 2334 3010 9903",
    ],
    test_negatives=[
        "0000000000000000",    # prefix 0 — not in any card BIN range (regex rejects \b4|5[1-5]|...)
        "9999999999999999",    # prefix 9 — not in any card BIN range
        "2345 6789 0126",      # 12 digits (Aadhaar format, not a card)
        "1234 5678 9012",      # prefix 1 — not in card range
        "ABCDE1234F",          # alpha chars — not digits
        "4532",                # too short (4 digits)
        "9876543210",          # 10 digits — too short for any card
        "1000000000000009",    # prefix 1 — not a valid BIN
        "0453201511283036",    # leading zero — word boundary blocks
        "3000000000000004",    # prefix 30 — not Amex (which is 34/37)
    ],
)

IN_DEBIT_CARD = PatternDef(
    id="IN_DEBIT_CARD",
    name="Debit Card Number",
    category="Financial Global",
    pii_category="Payment Card",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="critical",
    patterns=[
        # Same patterns as credit card — distinction is semantic/context-based
        r"\b4[0-9]{3}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}[\s\-]?[0-9]{1,4}\b",
        r"\b(?:5[1-5][0-9]{2}|2[2-7][0-9]{2})[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}\b",
        r"\b3[47][0-9]{2}[\s\-]?[0-9]{6}[\s\-]?[0-9]{5}\b",
        r"\b(?:6011|65[0-9]{2}|64[4-9][0-9])[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}\b",
        r"\b(?:60[0-9]{2}|65[0-9]{2}|8[12][0-9]{2}|508[0-9])[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}\b",
    ],
    flags=[],
    confidence_regex=0.60,
    confidence_validated=0.97,
    context_keywords=[
        "debit_card", "debit_card_no", "atm_card", "savings_card",
        "bank_card", "rupay_debit",
    ],
    validator=_luhn_validate,
    description="Debit card number with Luhn checksum validation. Same number space as credit cards.",
    false_positive_risk="medium",
    example="4532015112830366",
    country="GLOBAL",
    is_locked=True,
    notes="Separate from IN_CREDIT_CARD for DPDPA tagging — both use Luhn validation.",
    test_positives=[
        "4532015112830366",
        "4916338506082832",
        "5425233430109903",
        "2223000048400011",
        "371449635398431",
        "6011111111111117",
        "4532 0151 1283 0366",
        "4532-0151-1283-0366",
        "4916338506082832",
        "5425 2334 3010 9903",
    ],
    test_negatives=[
        "0000000000000000",    # prefix 0 — not a valid BIN
        "9999999999999999",    # prefix 9 — not a valid BIN
        "ABCDE1234F",          # alpha chars — not digits
        "2345 6789 0126",      # 12 digits (Aadhaar)
        "4532",                # too short (4 digits)
        "9876543210",          # 10 digits — too short
        "1000000000000009",    # prefix 1 — not a valid BIN
        "0453201511283036",    # leading zero — word boundary blocks
        "3000000000000004",    # prefix 30 — not Amex (34/37 only)
        "1234 5678 9012 3456", # prefix 1 — not in card range
    ],
)

IN_BANK_ACCOUNT = PatternDef(
    id="IN_BANK_ACCOUNT",
    name="Indian Bank Account Number",
    category="India Financial",
    pii_category="Bank Account",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="high",
    patterns=[
        # 9-18 digit number — base confidence 0.1; boosted by context
        r"(?<!\d)[0-9]{9,18}(?!\d)",
    ],
    flags=[],
    confidence_regex=0.10,
    confidence_validated=0.80,
    context_keywords=[
        "account_number", "acc_no", "bank_account", "acct_num",
        "a/c", "account_no", "savings_account", "current_account",
        "bank_acc", "acctno",
    ],
    validator=None,
    description="Indian bank account number (9-18 digits). Extremely low base confidence; context-gated.",
    false_positive_risk="critical",
    example="12345678901",
    country="IN",
    is_locked=True,
    notes="Without context, 9-18 digits matches almost anything. Requires 'account_number' column name.",
    test_positives=[
        "123456789",
        "1234567890",
        "12345678901",
        "123456789012",
        "1234567890123",
        "12345678901234",
        "123456789012345",
        "1234567890123456",
        "12345678901234567",
        "123456789012345678",
    ],
    test_negatives=[
        "12345678",               # 8 digits — too short (needs 9+)
        "1234567890123456789",    # 19 digits — too long (max 18)
        "ABCDE1234F",             # PAN — alpha chars
        "2345 6789 0126",         # Aadhaar — spaces break the digit run
        "100000/2024",            # has slash — not pure digits
        "Rs.12345678",            # has 'Rs.' prefix breaking digit run
        "0.12345678",             # decimal point breaks run
        "ACCT#123456-789",        # non-digit chars embedded
        "1234.567.890",           # dots break digit sequence
        "INV-20240101-001",       # invoice format — dashes break digits
    ],
)

IN_MICR = PatternDef(
    id="IN_MICR",
    name="MICR Code (Magnetic Ink Character Recognition)",
    category="India Financial",
    pii_category="Bank Branch Code",
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    sensitivity="medium",
    patterns=[
        # 9-digit MICR: city(3) + bank(3) + branch(3)
        r"(?<!\d)[0-9]{9}(?!\d)",
    ],
    flags=[],
    confidence_regex=0.50,
    confidence_validated=0.85,
    context_keywords=[
        "micr", "micr_code", "cheque_code", "cheque_number",
        "bank_micr", "branch_micr",
    ],
    validator=_micr_validate,
    description="9-digit MICR code on cheques: 3-digit city + 3-digit bank + 3-digit branch.",
    false_positive_risk="high",
    example="400240012",
    country="IN",
    is_locked=False,
    notes="Extremely broad pattern alone. Only reliable with 'micr' column context + format check.",
    test_positives=[
        "400240012",
        "110002001",
        "560001001",
        "600001001",
        "380001001",
        "411001001",
        "700012001",
        "500001001",
        "431001001",
        "302001001",
    ],
    test_negatives=[
        "12345678",         # 8 digits — too short (needs exactly 9)
        "1234567890",       # 10 digits — too long (needs exactly 9)
        "ABCDE1234F",       # PAN — alpha chars break the 9-digit pattern
        "400-240-012",      # hyphens — not 9 consecutive digits
        "HDFC0001234",      # IFSC — alpha prefix breaks lookbehind
        "400.240.012",      # dots break digit sequence
        "4002400120",       # 10 digits — too long
        "40024001",         # 8 digits — too short
        "MIC R400240",      # alpha chars
        "MICR: code",       # no 9-digit sequence
    ],
)

IN_SWIFT_BIC = PatternDef(
    id="IN_SWIFT_BIC",
    name="SWIFT / BIC Code",
    category="Financial Global",
    pii_category="Bank Identifier",
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    sensitivity="medium",
    patterns=[
        # 8 or 11 character BIC: 4-alpha bank + 2-alpha country + 2-alphanum location + optional 3-char branch
        r"(?i)\b[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}(?:[A-Z0-9]{3})?\b",
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.70,
    confidence_validated=0.95,
    context_keywords=[
        "swift", "bic", "swift_code", "bic_code", "correspondent_bank",
        "wire_transfer", "international_transfer",
    ],
    validator=_swift_validate,
    description="SWIFT/BIC code (8 or 11 chars) for international wire transfers.",
    false_positive_risk="medium",
    example="NWBKGB2L",
    country="GLOBAL",
    is_locked=False,
    notes="8-char BIC = head-office; 11-char = specific branch. Same format as IFSC but different length.",
    test_positives=[
        "NWBKGB2L",
        "DEUTDEDB",
        "BNPAFRPP",
        "CHASGB2L",
        "HDFCINBB",
        "SBININBB",
        "CITIINBX",
        "NWBKGB2LXXX",  # 11-char with branch
        "HDFCINBBXXX",
        "DEUTDEFFXXX",
    ],
    test_negatives=[
        "HDFC0001234",       # IFSC — 11 chars with 0 in pos 5 (IFSC, not BIC)
        "ABCDE1234F",        # PAN — 5 alpha + 4 digit + 1 alpha (not BIC)
        "NWBKGB",            # too short — only 6 chars (needs 8 or 11)
        "NWBKGB2LXXXXX",     # too long — 13 chars (max 11)
        "1WBKGB2L",          # starts with digit — regex requires alpha
        "ABC1GB2L",          # 3-char bank code (needs 4 alpha)
        "L17110MH1973",      # CIN prefix — 12 chars, not BIC format
        "27AAAPZ1234F",      # GSTIN — starts with digits
        "NW-BK-GB2L",        # hyphens break the run
        "NWBK12",            # only 6 chars — too short
    ],
)


# ─────────────────────────────────────────────────────────────────────────────
# SECTION 3: INDIA CONTACT DATA
# ─────────────────────────────────────────────────────────────────────────────

IN_MOBILE = PatternDef(
    id="IN_MOBILE",
    name="Indian Mobile Number",
    category="India Contact",
    pii_category="Phone Number",
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    sensitivity="medium",
    patterns=[
        # Format 1: +91 or 0091 + 10-digit mobile (high confidence)
        r"(?:\+91|0{2}91)[\s\-]?[6-9][0-9]{9}\b",
        # Format 2: 91XXXXXXXXXX without + (12 digits starting with 91)
        r"(?<!\d)91[6-9][0-9]{9}(?!\d)",
        # Format 3: 0XXXXXXXXXX (11 digits with leading 0)
        r"\b0[6-9][0-9]{9}\b",
        # Format 4: bare 10-digit mobile (context-gated, low base confidence)
        r"\b[6-9][0-9]{9}\b",
    ],
    flags=[],
    confidence_regex=0.50,
    confidence_validated=0.96,
    context_keywords=[
        "mobile", "phone", "contact", "cell", "telephone",
        "mobile_number", "phone_no", "mobile_no", "tel",
    ],
    validator=_mobile_validate,
    description="Indian mobile number in all 4 formats: +91, 91, 0, and bare 10-digit.",
    false_positive_risk="medium",
    example="+91 9876543210",
    country="IN",
    is_locked=True,
    notes=(
        "Format 4 (bare 10-digit) has confidence_regex=0.50 but full validator "
        "bumps to 0.96. Operator-range validation: first digit must be 6-9 "
        "(TRAI assigned ranges). All-same-digit and known dummy sequences rejected."
    ),
    test_positives=[
        "+91 9876543210",
        "+919876543210",
        "09876543210",
        "9876543210",
        "+91-8765432109",
        "0091 7654321098",
        "91 6543210987",
        "+91 7007007007",
        "08888888881",
        "9999999991",
    ],
    test_negatives=[
        "+91 0876543210",    # starts with 0 after country code — regex [6-9] rejects
        "+91 5876543210",    # starts with 5 — unassigned, regex [6-9] rejects
        "+91 1234567890",    # starts with 1 — regex [6-9] rejects
        "01234567890",       # leading 0 then 1 — \b0[6-9] rejects 01...
        "ABCDE1234F",        # alpha chars — no digit match
        "2345 6789 0126",    # Aadhaar — spaces between groups
        "123456789",         # 9 digits — too short for bare 10-digit pattern
        "+911234567890",     # starts with 1 after 91 — rejected by [6-9]
        "055-1234-5678",     # landline STD format, 0 then 5...
        "+1 2025551234",     # US country code with US number starting 2 — [6-9] rejects
    ],
)

IN_EMAIL = PatternDef(
    id="IN_EMAIL",
    name="Email Address",
    category="Global PII",
    pii_category="Email",
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    sensitivity="medium",
    patterns=[
        # RFC 5322 compliant, no catastrophic backtracking
        r"\b[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]{1,64}@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}\b",
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.80,
    confidence_validated=0.95,
    context_keywords=[
        "email", "email_address", "mail", "e_mail",
        "email_id", "contact_email",
    ],
    validator=_email_validate,
    description="RFC 5322 compliant email address. Not just .*@.* — validates structure.",
    false_positive_risk="low",
    example="user@example.com",
    country="GLOBAL",
    is_locked=True,
    notes="Uses non-backtracking pattern. Rejects consecutive dots and invalid TLDs.",
    test_positives=[
        "user@example.com",
        "user.name@domain.co.in",
        "user+tag@gmail.com",
        "firstname.lastname@company.org",
        "user123@domain.net",
        "test@test.io",
        "admin@company.co",
        "support@example.org",
        "noreply@hawk.in",
        "data.privacy@compliance.gov.in",
    ],
    test_negatives=[
        "notanemail",              # no @ sign
        "@nodomain.com",           # empty local part
        "user@",                   # no domain
        "user@.com",               # domain starts with dot
        "@",                       # empty everything
        "plainaddress",            # no @ sign
        "user@domain",             # no TLD (no dot after domain)
        "user@ domain.com",        # space after @ — space not in local/domain chars
        "user@domain..com",        # consecutive dots in domain
        "nodot",                   # no @ and no dot
    ],
)

IN_PINCODE = PatternDef(
    id="IN_PINCODE",
    name="Indian Postal Code (PIN)",
    category="India Contact",
    pii_category="Postal Code",
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    sensitivity="low",
    patterns=[
        # 6-digit, first digit 1-8 (no leading 0 or 9)
        r"\b[1-8][0-9]{5}\b",
    ],
    flags=[],
    confidence_regex=0.50,
    confidence_validated=0.85,
    context_keywords=[
        "pincode", "pin_code", "postal_code", "zip_code",
        "postcode", "zip", "area_code",
    ],
    validator=_pincode_validate,
    description="6-digit Indian PIN code (110001 Delhi to 855117 Kishanganj). Context-gated.",
    false_positive_risk="high",
    example="400001",
    country="IN",
    is_locked=False,
    notes="Range 110001-855117. Very common in addresses; needs pincode column context.",
    test_positives=[
        "110001",
        "400001",
        "560001",
        "600001",
        "700001",
        "380001",
        "302001",
        "500001",
        "226001",
        "110011",
    ],
    test_negatives=[
        "000001",       # starts with 0 — regex [1-8] rejects
        "900001",       # starts with 9 — regex [1-8] rejects
        "12345",        # 5 digits — too short
        "1234567",      # 7 digits — too long
        "900000",       # starts with 9 — regex [1-8] rejects
        "999999",       # starts with 9 — regex [1-8] rejects
        "Pin: ABC123",  # alpha chars — no 6-digit match
        "2345 6789",    # space breaks the run — not a 6-digit token
        "0110001",      # 7 digits starting with 0
        "pincode: --",  # no digits
    ],
)


# ─────────────────────────────────────────────────────────────────────────────
# SECTION 4: PERSONAL DEMOGRAPHICS
# ─────────────────────────────────────────────────────────────────────────────

IN_DOB = PatternDef(
    id="IN_DOB",
    name="Date of Birth",
    category="Personal",
    pii_category="Date of Birth",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="high",
    patterns=[
        # DD/MM/YYYY or DD-MM-YYYY or DD.MM.YYYY
        r"\b(?:0[1-9]|[12][0-9]|3[01])[/.\-](?:0[1-9]|1[0-2])[/.\-](?:19|20)\d{2}\b",
        # YYYY-MM-DD or YYYY/MM/DD
        r"\b(?:19|20)\d{2}[/.\-](?:0[1-9]|1[0-2])[/.\-](?:0[1-9]|[12][0-9]|3[01])\b",
        # DD-Mon-YYYY (01-Jan-1990)
        r"\b(?:0[1-9]|[12][0-9]|3[01])[\s\-](?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[\s\-](?:19|20)\d{2}\b",
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.70,
    confidence_validated=0.92,
    context_keywords=[
        "dob", "date_of_birth", "birth_date", "birthdate",
        "born_on", "date_birth", "birth_dt",
    ],
    validator=_dob_validate,
    description="Date of birth in DD/MM/YYYY, YYYY-MM-DD, and DD-Mon-YYYY formats.",
    false_positive_risk="medium",
    example="15/08/1990",
    country="GLOBAL",
    is_locked=False,
    notes="Year range 1900-2025 accepted. Validator checks calendar validity and age plausibility.",
    test_positives=[
        "15/08/1990",
        "01-01-1985",
        "31.12.2000",
        "1990-08-15",
        "2000/12/31",
        "01-Jan-1990",
        "15-Aug-1985",
        "28-Feb-2000",
        "31/12/1999",
        "01/01/2010",
    ],
    test_negatives=[
        "32/01/1990",      # invalid day (32) — regex (?:0[1-9]|[12][0-9]|3[01]) rejects 32
        "15/13/1990",      # invalid month (13) — regex (?:0[1-9]|1[0-2]) rejects 13
        "15/08/1890",      # year 18xx — regex (?:19|20) rejects 18xx
        "Order-15-08-90",  # 2-digit year — regex (?:19|20)\d{2} rejects
        "Invoice 15082024",# no separators — regex requires [/.\-]
        "Tax: 15%",        # no date format
        "2024-00-01",      # month 00 — regex (?:0[1-9]|1[0-2]) rejects 00
        "00/08/1990",      # day 00 — regex (?:0[1-9]|...) rejects 00
        "1800-01-01",      # year 1800 — regex (?:19|20) rejects
        "abc-def-ghij",    # no numeric date
    ],
)

IN_AGE_NUMERIC = PatternDef(
    id="IN_AGE_NUMERIC",
    name="Age (Numeric, Context-Window)",
    category="Personal",
    pii_category="Age",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="high",
    patterns=[
        # "age: N" or "age: NN" in context window
        r"(?i)\bage[\s:=]+([0-9]{1,3})\b",
        # "aged N years" or "N years old"
        r"(?i)\baged?\s+([0-9]{1,3})\s+years?\b",
        r"(?i)\b([0-9]{1,3})\s+years?\s+old\b",
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.70,
    confidence_validated=0.88,
    context_keywords=[
        "age", "patient_age", "age_years", "age_at_admission",
        "age_on_date",
    ],
    validator=None,
    description="Numeric age indicator in context window: 'age: 16', 'aged 23 years', '45 years old'.",
    false_positive_risk="medium",
    example="age: 25",
    country="GLOBAL",
    is_locked=False,
    notes="Context-window mode only. Standalone numbers without 'age' keyword are not matched.",
    test_positives=[
        "age: 25",
        "age: 7",
        "age: 120",
        "age = 45",
        "aged 30 years",
        "aged 5 years",
        "45 years old",
        "23 year old",
        "age:18",
        "Age: 65",
    ],
    test_negatives=[
        "25",              # bare number — no age/aged/years keyword
        "page: 25",        # 'page' ≠ 'age' — word boundary \bage prevents match
        "stage: 3",        # 'stage' ≠ 'age' — \bage requires word start
        "usage: 25",       # 'usage' ends with 'age' but \bage needs word boundary
        "dosage: 25mg",    # 'dosage' ≠ standalone \bage
        "percentage: 25",  # no age keyword
        "year 2025",       # 'year' without preceding number
        "file age: old",   # 'old' not a number — no digits after age:
        "age group: adult",# no numeric value — 'adult' not digits
        "years experience", # no number before 'years'
    ],
)

IN_BLOOD_GROUP = PatternDef(
    id="IN_BLOOD_GROUP",
    name="Blood Group / Type",
    category="Personal",
    pii_category="Health Data",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="high",
    patterns=[
        # ABO + Rh system — lookahead prevents matching A+B (only A+ at word/end)
        r"(?i)\b(?:AB|A|B|O)[+\-](?![a-zA-Z0-9])",
        # Spelled out
        r"(?i)\b(?:A|B|AB|O)\s*(?:positive|negative|pos|neg)\b",
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.70,
    confidence_validated=0.90,
    context_keywords=[
        "blood_group", "blood_type", "blood", "rh_factor", "abo_type",
    ],
    validator=None,
    description="ABO blood group and Rh factor (e.g. A+, O-, AB positive).",
    false_positive_risk="medium",
    example="O+",
    country="GLOBAL",
    is_locked=False,
    notes="Requires context column (blood_group, blood_type) to reduce FP rate.",
    test_positives=[
        "A+",
        "B-",
        "AB+",
        "O-",
        "A positive",
        "B negative",
        "AB positive",
        "O negative",
        "A pos",
        "B neg",
    ],
    test_negatives=[
        "C+",           # C is not a blood type
        "D+",           # D is not a valid ABO type
        "A grade",      # grade not blood type (no +/- after A)
        "A sector",     # no +/- after A
        "A+B",          # lookahead (?![a-zA-Z0-9]) blocks — B follows +
        "O(1)",         # function call — ( doesn't match [+\-]
        "B2 vitamin",   # B followed by digit not +/-
        "Option A",     # A with no following +/-
        "Version A1",   # digit follows A
        "class A",      # A with no following +/-
    ],
)

IN_GENDER = PatternDef(
    id="IN_GENDER",
    name="Gender / Sex Indicator",
    category="Personal",
    pii_category="Gender",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="high",
    patterns=[
        r"(?i)\b(?:Male|Female|M|F|Other|Non[\s\-]?binary|Prefer not to say|Unknown|Transgender|Intersex)\b",
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.75,
    confidence_validated=0.90,
    context_keywords=[
        "gender", "sex", "biological_sex", "gender_identity", "sex_at_birth",
    ],
    validator=None,
    description="Gender or biological sex value in a column named gender/sex.",
    false_positive_risk="medium",
    example="Female",
    country="GLOBAL",
    is_locked=False,
    notes="Context-gated — standalone M/F chars are extremely common. Only fire with gender column.",
    test_positives=[
        "Male",
        "Female",
        "M",
        "F",
        "Other",
        "Non-binary",
        "Non binary",
        "Prefer not to say",
        "Transgender",
        "Unknown",
    ],
    test_negatives=[
        "Mail",           # mail not male — not in alternation
        "FM radio",       # FM abbreviation — not in alternation
        "Mr",             # title — not in alternation
        "Mrs",            # title — not in alternation
        "Form",           # not in alternation
        "Manufacturing",  # not in alternation
        "Finland",        # not in alternation
        "Feature",        # not in alternation
        "Mango",          # not in alternation
        "Football",       # not in alternation
    ],
)


# ─────────────────────────────────────────────────────────────────────────────
# SECTION 5: SENSITIVE PERSONAL DATA (DPDPA Schedule II)
# ─────────────────────────────────────────────────────────────────────────────

IN_BIOMETRIC_REF = PatternDef(
    id="IN_BIOMETRIC_REF",
    name="Biometric Reference Token",
    category="India Healthcare",
    pii_category="Biometric Data",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="critical",
    patterns=[
        # Biometric template/token: BREF- or BIO- prefix + hex/b64 string
        r"(?i)\b(?:BREF|BIO|BIOMETRIC|FPRINT|IRIS|FACE|RETINA)[\s\-_]?[A-Z0-9]{16,64}\b",
        # UIDAI biometric token format (base64-like, 40+ chars in biometric context)
        r"(?i)(?:biometric|fingerprint|iris|facial|face_id)[\s=:\"']+([A-Za-z0-9+/=]{20,100})\b",
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.70,
    confidence_validated=0.90,
    context_keywords=[
        "biometric", "fingerprint", "iris_scan", "face_id",
        "retina_scan", "bio_token", "biometric_id",
    ],
    validator=None,
    description="Biometric reference token or encoded biometric data field.",
    false_positive_risk="medium",
    example="BREF-A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4",
    country="GLOBAL",
    is_locked=False,
    notes="No mathematical validator exists for biometric tokens. Context-window mode required.",
    test_positives=[
        "BREF-A1B2C3D4E5F6A1B2C3D4E5F6A1B2",
        "BIO-1234567890ABCDEF1234567890ABCDEF",
        "FPRINT-ABCDEF1234567890ABCDEF12345678",
        "IRIS-1234567890123456789012345678901234",
        "FACE-ABCDEFABCDEFABCDEFABCDEFABCDEF12",
        "biometric: SGVsbG9Xb3JsZA==YWJjZGVmZ2g=",
        "fingerprint: dGVzdGZpbmdlcnByaW50AAAA",
        "iris: aXJpc3NjYW5kYXRhZXhhbXBsZQ==",
        "face_id: ZmFjZWlkZGF0YXNhbXBsZWZpbGU=",
        "BIOMETRIC-TOKENABC123DEF456GHI789JKL",
    ],
    test_negatives=[
        "biometric: unknown",    # too short value
        "BREF",                  # prefix alone
        "fingerprint: N/A",
        "AWS_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE",
        "API_KEY=AIzaSyDaGmWKa4JsXZ",
        "BIO123",                # too short after prefix
        "iris: blue",            # color not token
        "fingerprint scanner",   # description not data
        "face recognition",      # description
        "bio: 123",              # too short
    ],
)

IN_MRN = PatternDef(
    id="IN_MRN",
    name="Medical Record Number / UHID",
    category="India Healthcare",
    pii_category="Medical Record",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="critical",
    patterns=[
        # MRN/UHID: usually alphanumeric, hospital-specific format
        r"(?i)\b(?:MRN|UHID|MR[\s\-]?NO|MED[\s\-]?REC|PATID|HRN)[\s:=\-]+[A-Z0-9]{5,15}\b",
        # Numeric MRN with context keyword
        r"(?i)(?:mrn|uhid|medical_record|patient_id)[\s:=\-\"']+([A-Z0-9]{5,15})\b",
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.75,
    confidence_validated=0.90,
    context_keywords=[
        "mrn", "uhid", "medical_record", "patient_id", "hospital_id",
        "medical_record_number", "patient_record",
    ],
    validator=None,
    description="Medical Record Number (MRN) or Universal Health ID (UHID) — hospital-specific format.",
    false_positive_risk="medium",
    example="MRN-123456",
    country="IN",
    is_locked=False,
    notes="Format varies by hospital. Context keyword (MRN/UHID) required for reliable detection.",
    test_positives=[
        "MRN-123456",
        "UHID: ABC12345",
        "MR NO: 1234567",
        "MED-REC-ABC123",
        "PATID: P001234",
        "HRN: HR123456",
        "mrn: 12345ABC",
        "medical_record: MRN00123",
        "patient_id: PAT12345",
        "uhid: UHID12345",
    ],
    test_negatives=[
        "MR. Smith",         # title abbreviation
        "MR. Jones",
        "hr@company.com",    # HR email
        "HR Department",     # HR abbreviation
        "MRN",               # prefix without number
        "UHID",              # prefix without number
        "Order ID: 12345",   # order ID
        "Invoice: 12345",
        "Patient admitted",  # no ID
        "Room: 201",         # room number
    ],
)

IN_CTRI = PatternDef(
    id="IN_CTRI",
    name="Clinical Trial Registration Number (CTRI)",
    category="India Healthcare",
    pii_category="Clinical Data",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="high",
    patterns=[
        # CTRI/YYYY/MM/NNNNNN format
        r"(?i)\bCTRI/[0-9]{4}/[0-9]{2}/[0-9]{6}\b",
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.90,
    confidence_validated=0.98,
    context_keywords=[
        "ctri", "clinical_trial", "trial_registration", "trial_id",
        "clinical_study",
    ],
    validator=_ctri_validate,
    description="Clinical Trial Registry of India number: CTRI/YYYY/MM/NNNNNN.",
    false_positive_risk="low",
    example="CTRI/2021/01/030296",
    country="IN",
    is_locked=False,
    notes="Very distinctive prefix. Low FP risk.",
    test_positives=[
        "CTRI/2021/01/030296",
        "CTRI/2019/06/019876",
        "CTRI/2022/12/054321",
        "CTRI/2020/03/041234",
        "CTRI/2018/07/012345",
        "ctri/2021/01/030296",  # lowercase
        "CTRI/2023/09/067890",
        "CTRI/2017/02/007654",
        "CTRI/2024/11/089012",
        "CTRI/2016/04/003456",
    ],
    test_negatives=[
        "CTRI",                   # prefix alone — no slash follows
        "CTRI/2021",              # incomplete — no /MM/NNNNNN
        "CTRI/2021/01",           # incomplete — no /NNNNNN
        "CTRI/21/01/030296",      # 2-digit year — `[0-9]{4}` needs exactly 4 digits
        "NCT00123456",            # US format — starts with NCT not CTRI
        "EUCTR2021-001234-15-DE", # EU CTR format — hyphens not slashes
        "Invoice CTRI/001",       # no slash-separated 4+2+6 after CTRI
        "Registration: 030296",   # no CTRI prefix
        "Study Protocol v2.1",    # no CTRI format
        "CTRI-2021-01-030296",    # hyphens instead of slashes — won't match /[0-9]{4}/...
    ],
)

IN_CASTE_INDICATOR = PatternDef(
    id="IN_CASTE_INDICATOR",
    name="Caste Category Indicator",
    category="India Sensitive",
    pii_category="Caste",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="critical",
    patterns=[
        # Explicit caste category codes in India
        r"(?i)\b(?:SC|ST|OBC|General|EWS|VJNT|NT[1-4]|SBC|DT)\b",
        # Context: caste followed by category
        r"(?i)\bcaste[\s:=\-\"']+([A-Za-z][A-Za-z\s]{1,39})\b",
        # Specific caste mentions — require explicit caste-specific column keywords
        r"(?i)(?:caste_category|caste_group|reservation_category|social_category)[\s:=\-\"']+([A-Za-z][A-Za-z\s]{1,39})\b",
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.60,
    confidence_validated=0.85,
    context_keywords=[
        "caste", "caste_category", "community", "reservation_category",
        "category_code", "social_category",
    ],
    validator=None,
    description="Caste category indicator (SC/ST/OBC/General/EWS) — DPDPA Schedule II sensitive data.",
    false_positive_risk="high",
    example="SC",
    country="IN",
    is_locked=False,
    notes="SC/ST/OBC codes are extremely common abbreviations. Must have caste/category column context.",
    test_positives=[
        "SC",
        "ST",
        "OBC",
        "General",
        "EWS",
        "caste: SC",
        "caste: Brahmin",
        "caste_category: OBC",
        "caste_group: ST",
        "reservation_category: EWS",
    ],
    test_negatives=[
        "blood_group: O+",      # no caste keyword
        "product_id: AB123",    # no caste keyword
        "language: English",    # no caste keyword
        "country: India",       # no caste keyword
        "department: Finance",  # no caste keyword
        "amount: 1000",         # no caste keyword
        "ticket_status: open",  # no caste keyword — 'status' ≠ caste keywords
        "server: running",      # no caste keyword
        "version: 2.0",         # no caste keyword
        "priority: high",       # no caste keyword
    ],
)

IN_RELIGION_INDICATOR = PatternDef(
    id="IN_RELIGION_INDICATOR",
    name="Religion Indicator",
    category="India Sensitive",
    pii_category="Religious Belief",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="critical",
    patterns=[
        # Major Indian religions
        r"(?i)\b(?:Hindu|Muslim|Christian|Sikh|Buddhist|Jain|Jewish|Parsi|Bahai|Tribal)\b",
        # Context: religion followed by value — require value to start with a letter
        r"(?i)\breligion[\s:=\-\"']+([A-Za-z][A-Za-z\s]{1,39})\b",
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.55,
    confidence_validated=0.85,
    context_keywords=[
        "religion", "faith", "religious_affiliation", "religion_code",
        "community_religion",
    ],
    validator=None,
    description="Religious affiliation indicator — DPDPA Schedule II sensitive personal data.",
    false_positive_risk="high",
    example="Hindu",
    country="IN",
    is_locked=False,
    notes="Religion names appear in many non-PII contexts. Context column required.",
    test_positives=[
        "Hindu",
        "Muslim",
        "Christian",
        "Sikh",
        "Buddhist",
        "Jain",
        "Parsi",
        "religion: Hindu",
        "religion: Muslim",
        "faith: Sikh",
    ],
    test_negatives=[
        "blood_group: O+",      # no religion keyword
        "language: Tamil",      # no religion keyword
        "country: India",       # no religion keyword
        "department: HR",       # no religion keyword
        "religion: N/A",        # 'N/A' — value starts with 'N' then '/' breaks the match
        "religion: --",         # no letter-starting value after keyword
        "profession: doctor",   # no religion keyword
        "age: 35",              # no religion keyword
        "status: active",       # no religion keyword
        "address: Mumbai",      # no religion keyword
    ],
)

IN_POLITICAL_AFFILIATION = PatternDef(
    id="IN_POLITICAL_AFFILIATION",
    name="Political Opinion / Party Affiliation",
    category="India Sensitive",
    pii_category="Political Opinion",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="critical",
    patterns=[
        # Major Indian political parties
        r"(?i)\b(?:BJP|INC|AAP|SP|BSP|TMC|DMK|AIDMK|NCP|CPI|CPM|RJD|JDU|TRS|YSRCP|BJD)\b",
        # Context: political affiliation/party followed by value
        r"(?i)\b(?:political_party|party_affiliation|party_member)[\s:=\-\"']+([A-Za-z\s]{2,40})\b",
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.50,
    confidence_validated=0.80,
    context_keywords=[
        "political_party", "party", "party_affiliation",
        "political_opinion", "party_membership",
    ],
    validator=None,
    description="Political party affiliation or political opinion — DPDPA Schedule II sensitive data.",
    false_positive_risk="high",
    example="BJP",
    country="IN",
    is_locked=False,
    notes="Party abbreviations (BJP, AAP) are common in news text. Must have political_party column context.",
    test_positives=[
        "BJP",
        "INC",
        "AAP",
        "political_party: BJP",
        "party_affiliation: INC",
        "party_member: AAP",
        "SP",
        "BSP",
        "TMC",
        "DMK",
    ],
    test_negatives=[
        "party: Birthday",              # party alone not in keyword list (needs party_affiliation)
        "party_type: Formal",           # party_type not in keyword list
        "NDA coalition",                # NDA not in party abbreviation list
        "UPA government",               # UPA not in list
        "GDP growth rate",              # no political party keyword
        "political_affiliation: None",  # 'None' starts alpha — matches! fix: use number
        "team: Backend",                # no political keyword
        "country: India",               # no political keyword
        "role: developer",              # no political keyword
        "sector: technology",           # no political keyword
    ],
)


# ─────────────────────────────────────────────────────────────────────────────
# SECTION 6: MULTI-LANGUAGE VARIANTS
# ─────────────────────────────────────────────────────────────────────────────

IN_AADHAAR_DEVANAGARI = PatternDef(
    id="IN_AADHAAR_DEVANAGARI",
    name="Aadhaar Number (Devanagari Context)",
    category="India Identity",
    pii_category="Unique Identifier",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="critical",
    patterns=[
        # Devanagari digits (Unicode range U+0966–U+096F) in 12-digit pattern
        r"[\u0966-\u096F]{4}[\s\u200C\u200D]?[\u0966-\u096F]{4}[\s\u200C\u200D]?[\u0966-\u096F]{4}",
        # Mixed ASCII + Devanagari labels: "आधार:" or "आधार नंबर" or "UIDAI" followed by digits
        # Allow any mix of whitespace, punctuation, and Devanagari text between keyword and number
        r"(?:आधार|आधार\s+संख्या|आधार\s+नंबर|UIDAI?|UID)[\s:=\u0900-\u097F]{1,50}[2-9][0-9]{3}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}",
    ],
    flags=[],
    confidence_regex=0.75,
    confidence_validated=0.95,
    context_keywords=[
        "aadhaar", "आधार", "uid", "uidai", "aadhar",
    ],
    validator=_aadhaar_validate,
    description="Aadhaar number appearing in Hindi/Marathi/Devanagari script context.",
    false_positive_risk="low",
    example="आधार: 2345 6789 0126",
    country="IN",
    is_locked=False,
    notes="Devanagari numerals (०१२३...) normalized to ASCII before Verhoeff validation.",
    test_positives=[
        "आधार: 2345 6789 0126",
        "आधार संख्या: 234567890126",
        "आधार नंबर: 5234-5678-9015",
        "UID: 9876 5432 1095",
        "आधार:234567890126",
        "\u0966\u0967\u0968\u0969\u096a\u096b\u096c\u096d\u096e\u096f\u0966\u0967",  # Devanagari digits
        "आधार: 8765 4321 0987",
        "आधार: 6543 2109 8765",
        "UIDAI नंबर: 4321 0987 6543",
        "आधार: 7890 1234 5670",
    ],
    test_negatives=[
        "आधार कार्ड",              # "Aadhaar card" — no number
        "आधार: 123",               # too short
        "आधार: ABCD1234",          # not digits
        "पैन: ABCDE1234F",         # PAN, not Aadhaar
        "मोबाइल: 9876543210",       # mobile number
        "पिन: 110001",             # PIN code
        "दिनांक: 15/08/1990",       # date
        "भार: 75",                  # "weight: 75" — not Aadhaar
        "आधार: 1234 5678 9012",    # starts with 1 (invalid Aadhaar)
        "पता: मुंबई",               # address, no number
    ],
)

IN_MOBILE_DEVANAGARI = PatternDef(
    id="IN_MOBILE_DEVANAGARI",
    name="Indian Mobile Number (Devanagari Context)",
    category="India Contact",
    pii_category="Phone Number",
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    sensitivity="medium",
    patterns=[
        # "मोबाइल:" or "फोन:" followed by 10-digit number
        r"(?:मोबाइल|फोन|संपर्क|दूरभाष)[\s:=]+(?:\+91[\s\-]?)?[6-9][0-9]{9}\b",
        # Devanagari digits for mobile: starts with ६-९
        r"(?:मोबाइल|फोन)[\s:=]+[\u096C-\u096F][\u0966-\u096F]{9}",
    ],
    flags=[],
    confidence_regex=0.75,
    confidence_validated=0.92,
    context_keywords=[
        "मोबाइल", "फोन", "mobile", "phone", "संपर्क",
    ],
    validator=_mobile_validate,
    description="Indian mobile number appearing in Hindi/Devanagari script context.",
    false_positive_risk="low",
    example="मोबाइल: 9876543210",
    country="IN",
    is_locked=False,
    notes="Devanagari numerals normalized to ASCII for validation.",
    test_positives=[
        "मोबाइल: 9876543210",
        "फोन: +91 8765432109",
        "संपर्क: 7654321098",
        "दूरभाष: 9876543210",
        "मोबाइल: +91-6543210987",
        "फोन:9999999991",
        "मोबाइल: 8888888882",
        "संपर्क: 7777777773",
        "फोन: 6666666664",
        "मोबाइल: 9123456789",
    ],
    test_negatives=[
        "मोबाइल: 1234567890",    # starts with 1 — regex [6-9] first digit rejects
        "मोबाइल: 5876543210",    # starts with 5 — regex [6-9] rejects
        "फोन: 12345",            # too short — 5 digits
        "आधार: 234567890126",    # no Devanagari phone keyword
        "पिन: 110001",           # no phone keyword
        "मोबाइल: ABCDE1234F",    # alpha chars — no digit match
        "मोबाइल",               # label with no number
        "संख्या: 0",             # zero — no phone keyword
        "दूरभाष: +1 9876543210", # +1 then 1... regex needs [6-9] after prefix
        "नाम: राहुल",            # name label — no number
    ],
)


# ─────────────────────────────────────────────────────────────────────────────
# SECTION 7: GLOBAL PII PATTERNS (KEY ONES)
# ─────────────────────────────────────────────────────────────────────────────

GLOBAL_EMAIL = PatternDef(
    id="GLOBAL_EMAIL",
    name="Email Address (Global)",
    category="Global PII",
    pii_category="Email",
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    sensitivity="medium",
    patterns=[
        r"\b[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]{1,64}@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}\b",
    ],
    flags=["IGNORECASE"],
    confidence_regex=0.80,
    confidence_validated=0.95,
    context_keywords=["email", "email_address", "mail", "e_mail", "contact_email"],
    validator=_email_validate,
    description="RFC 5322-compliant email address (global).",
    false_positive_risk="low",
    example="user@example.com",
    country="GLOBAL",
    is_locked=True,
    notes="Duplicate of IN_EMAIL but explicitly GLOBAL-scoped for cross-border use.",
    test_positives=[
        "user@example.com",
        "user.name@domain.co.uk",
        "user+tag@gmail.com",
        "test@test.io",
        "admin@company.org",
        "support@example.net",
        "noreply@hawk.com",
        "data@compliance.gov.in",
        "a@b.co",
        "user123@sub.domain.example.com",
    ],
    test_negatives=[
        "notanemail",              # no @ sign
        "@nodomain.com",           # empty local part
        "user@",                   # no domain
        "user@.com",               # domain starts with dot
        "@",                       # empty everything
        "user@domain",             # no TLD (no dot after domain)
        "user@ domain.com",        # space after @ — invalid
        "user@domain..com",        # consecutive dots in domain
        "nodot",                   # no @ or dot
        "missing-at-sign.com",     # no @ sign
    ],
)

GLOBAL_CREDIT_CARD = PatternDef(
    id="GLOBAL_CREDIT_CARD",
    name="Credit Card Number (Global)",
    category="Financial Global",
    pii_category="Payment Card",
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    sensitivity="critical",
    patterns=[
        r"\b4[0-9]{3}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}[\s\-]?[0-9]{1,4}\b",
        r"\b(?:5[1-5][0-9]{2}|2[2-7][0-9]{2})[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}\b",
        r"\b3[47][0-9]{2}[\s\-]?[0-9]{6}[\s\-]?[0-9]{5}\b",
        r"\b(?:6011|65[0-9]{2}|64[4-9][0-9])[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}\b",
    ],
    flags=[],
    confidence_regex=0.60,
    confidence_validated=0.97,
    context_keywords=["credit_card", "card_number", "cc_number", "visa", "mastercard", "amex"],
    validator=_luhn_validate,
    description="Credit card number (global) with Luhn validation.",
    false_positive_risk="medium",
    example="4532015112830366",
    country="GLOBAL",
    is_locked=True,
    notes="Global version without RuPay. See IN_CREDIT_CARD for India-specific with RuPay.",
    test_positives=[
        "4532015112830366",
        "4916338506082832",
        "5425233430109903",
        "2223000048400011",
        "374251018720955",
        "6011111111111117",
        "4532 0151 1283 0366",
        "4532-0151-1283-0366",
        "4916338506082832",
        "5425 2334 3010 9903",
    ],
    test_negatives=[
        "1234567890123456",    # prefix 1 — not in any card BIN range
        "0000000000000000",    # prefix 0 — not valid
        "9999999999999999",    # prefix 9 — not valid
        "2345 6789 0126",      # 12 digits — Aadhaar format
        "ABCDE1234F",          # alpha chars — not a card number
        "27AAAPZ1234F1Z5",     # GSTIN — starts with digits but wrong format
        "Order123456789",      # only 9 digits after 'Order'
        "1111111111111111",    # prefix 1 — not valid BIN
        "0453201511283036",    # leading zero — word boundary blocks
        "3000000000000004",    # prefix 30 — not Amex (34/37 required)
    ],
)


# ─────────────────────────────────────────────────────────────────────────────
# SECTION 8: MASTER REGISTRY
# ─────────────────────────────────────────────────────────────────────────────
#
# ALL_PATTERNS is the single source of truth for every pattern in the system.
# Custom patterns from the DB are appended at runtime (not stored here).
#
# To add a new pattern:
#   1. Define it as a PatternDef above
#   2. Add it to the appropriate section list below
#   3. Run hawk_patterns_test.py — all test_positives must pass, test_negatives must fail
#   4. Submit PR with benchmark results
#

# India Government Identity
_INDIA_IDENTITY = [
    IN_AADHAAR,
    IN_PAN,
    IN_PASSPORT,
    IN_VOTER_ID,
    IN_DRIVING_LICENSE,
    IN_GSTIN,
    IN_CIN,
    IN_IFSC,
    IN_UPI,
    IN_RATION_CARD,
    IN_UAN,
    IN_ABHA,
    IN_AADHAAR_DEVANAGARI,
]

# India Financial
_INDIA_FINANCIAL = [
    IN_CREDIT_CARD,
    IN_DEBIT_CARD,
    IN_BANK_ACCOUNT,
    IN_MICR,
    IN_SWIFT_BIC,
]

# India Contact
_INDIA_CONTACT = [
    IN_MOBILE,
    IN_EMAIL,
    IN_PINCODE,
    IN_MOBILE_DEVANAGARI,
]

# India Personal Demographics
_INDIA_PERSONAL = [
    IN_DOB,
    IN_AGE_NUMERIC,
    IN_BLOOD_GROUP,
    IN_GENDER,
]

# India Sensitive (DPDPA Schedule II)
_INDIA_SENSITIVE = [
    IN_BIOMETRIC_REF,
    IN_MRN,
    IN_CTRI,
    IN_CASTE_INDICATOR,
    IN_RELIGION_INDICATOR,
    IN_POLITICAL_AFFILIATION,
]

# Global
_GLOBAL = [
    GLOBAL_EMAIL,
    GLOBAL_CREDIT_CARD,
]

# Combined ALL_PATTERNS registry (insertion order maintained in Python 3.7+)
def _build_registry() -> Dict[str, PatternDef]:
    all_sections = [
        _INDIA_IDENTITY,
        _INDIA_FINANCIAL,
        _INDIA_CONTACT,
        _INDIA_PERSONAL,
        _INDIA_SENSITIVE,
        _GLOBAL,
    ]
    registry: Dict[str, PatternDef] = {}
    for section in all_sections:
        for pat in section:
            if pat.id in registry:
                raise ValueError(f"Duplicate pattern ID: {pat.id!r}")
            registry[pat.id] = pat
    return registry


ALL_PATTERNS: Dict[str, PatternDef] = _build_registry()


# ─────────────────────────────────────────────────────────────────────────────
# PUBLIC API
# ─────────────────────────────────────────────────────────────────────────────

def get_pattern(pattern_id: str) -> Optional[PatternDef]:
    """Look up a pattern by ID (case-insensitive)."""
    return ALL_PATTERNS.get(pattern_id.upper().strip())


def get_by_category(category: str) -> List[PatternDef]:
    """Return all patterns in a given category (case-insensitive)."""
    cat = category.lower()
    return [p for p in ALL_PATTERNS.values() if p.category.lower() == cat]


def get_by_sensitivity(sensitivity: str) -> List[PatternDef]:
    """Return all patterns at a given sensitivity level."""
    level = sensitivity.lower()
    return [p for p in ALL_PATTERNS.values() if p.sensitivity == level]


def get_by_dpdpa_schedule(schedule: str) -> List[PatternDef]:
    """Return all patterns under a given DPDPA schedule."""
    return [p for p in ALL_PATTERNS.values() if p.dpdpa_schedule == schedule]


def get_indian_pii_patterns() -> List[PatternDef]:
    """Return all India-specific PII patterns (country='IN')."""
    return [p for p in ALL_PATTERNS.values() if p.country == "IN"]


def get_stats() -> dict:
    """Return summary statistics about the registry."""
    from collections import Counter
    total = len(ALL_PATTERNS)
    locked = sum(1 for p in ALL_PATTERNS.values() if p.is_locked)
    by_cat = Counter(p.category for p in ALL_PATTERNS.values())
    by_sens = Counter(p.sensitivity for p in ALL_PATTERNS.values())
    by_schedule = Counter(p.dpdpa_schedule for p in ALL_PATTERNS.values())
    with_validator = sum(1 for p in ALL_PATTERNS.values() if p.validator is not None)
    with_tests = sum(
        1 for p in ALL_PATTERNS.values()
        if len(p.test_positives) >= 10 and len(p.test_negatives) >= 10
    )
    return {
        "total": total,
        "locked": locked,
        "with_validator": with_validator,
        "with_full_test_suite": with_tests,
        "by_category": dict(by_cat),
        "by_sensitivity": dict(by_sens),
        "by_dpdpa_schedule": dict(by_schedule),
    }


def run_startup_tests() -> bool:
    """
    Run all pattern self-tests at startup.

    Returns:
        True if ALL tests pass, False if any fail.
        Prints a summary to stdout.
    """
    from hawk_validators import backtracking_safe

    total_patterns = 0
    total_tests = 0
    failures = []

    for pat in ALL_PATTERNS.values():
        total_patterns += 1
        compiled = pat.compile()

        # Backtracking safety test
        for pattern_str in pat.patterns:
            if not backtracking_safe(pattern_str):
                failures.append(f"[BACKTRACK] {pat.id}: pattern '{pattern_str[:50]}' fails safety test")

        # Positive tests
        for sample in pat.test_positives:
            total_tests += 1
            matched = any(c.search(sample) for c in compiled)
            if not matched:
                failures.append(f"[FALSE_NEG] {pat.id}: '{sample}' should match but didn't")

        # Negative tests
        for sample in pat.test_negatives:
            total_tests += 1
            matched = any(c.search(sample) for c in compiled)
            if matched:
                failures.append(f"[FALSE_POS] {pat.id}: '{sample}' should NOT match but did")

    pass_count = total_tests - len(failures)
    print(f"\nHawk Pattern Self-Test: {total_patterns} patterns, {total_tests} tests")
    print(f"  PASSED: {pass_count}/{total_tests}")
    if failures:
        print(f"  FAILED: {len(failures)}")
        for f in failures[:20]:  # Show first 20 failures
            print(f"    {f}")
        if len(failures) > 20:
            print(f"    ... and {len(failures) - 20} more")
        return False
    else:
        print("  ALL TESTS PASSED")
        return True


if __name__ == "__main__":
    stats = get_stats()
    print("=== Hawk Pattern Library ===")
    print(f"Total patterns: {stats['total']}")
    print(f"Locked:         {stats['locked']}")
    print(f"With validator: {stats['with_validator']}")
    print(f"Full test suite (≥10 pos + ≥10 neg): {stats['with_full_test_suite']}")
    print("\nBy sensitivity:")
    for s, c in sorted(stats["by_sensitivity"].items()):
        print(f"  {s:<12} {c}")
    print("\nBy DPDPA schedule:")
    for s, c in sorted(stats["by_dpdpa_schedule"].items()):
        print(f"  {s:<35} {c}")
    print()
    run_startup_tests()
