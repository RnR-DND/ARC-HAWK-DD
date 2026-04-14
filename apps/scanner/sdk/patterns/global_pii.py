"""
Global PII Patterns
====================
International PII patterns — US, UK, EU, AU, CA, SG, MY.

Covered:
  US_SSN         US Social Security Number
  US_EIN         US Employer Identification Number
  US_PASSPORT    US Passport Number
  UK_NHS         UK National Health Service number (mod-11)
  UK_NI          UK National Insurance number
  UK_UTR         UK Unique Taxpayer Reference
  AU_TFN         Australian Tax File Number (mod-11)
  CA_SIN         Canadian Social Insurance Number (Luhn)
  SG_NRIC        Singapore NRIC / FIN
  MY_NRIC        Malaysia MyKad / NRIC
  EU_VAT_DE      German VAT number
  EU_VAT_FR      French VAT number
  EU_VAT_GB      UK VAT number
  EU_VAT_IT      Italian VAT number
  EU_VAT_ES      Spanish VAT number
"""

import re
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from .base import PiiPattern, PatternCategory, DPDPASchedule


# --------------------------------------------------------------------------- #
# Validators                                                                   #
# --------------------------------------------------------------------------- #

def _validate_us_ssn(value: str) -> bool:
    """
    US SSN basic format validation.
    Rejects: area 000, 666, 900-999 (ITIN); group 00; serial 0000.
    Example valid: 123-45-6789
    """
    clean = re.sub(r"[\s-]", "", value)
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


def _validate_uk_nhs(value: str) -> bool:
    """
    UK NHS number: 10-digit, weighted mod-11.
    Weights: 10, 9, 8, 7, 6, 5, 4, 3, 2 for positions 1-9.
    Sum + check_digit must be divisible by 11; check digit 10 is invalid.

    Known valid: 943-476-5919
    """
    clean = re.sub(r"[\s-]", "", value)
    if not re.match(r"^\d{10}$", clean):
        return False
    weights = [10, 9, 8, 7, 6, 5, 4, 3, 2]
    total = sum(int(clean[i]) * weights[i] for i in range(9))
    remainder = total % 11
    check = 11 - remainder if remainder != 0 else 0
    if check == 10:
        return False
    return check == int(clean[9])


def _validate_au_tfn(value: str) -> bool:
    """
    Australian TFN: 8 or 9 digits, weighted mod-11.
    9-digit weights: 1, 4, 3, 7, 5, 8, 6, 9, 10.
    8-digit weights: 10, 7, 8, 4, 6, 3, 5, 1.
    Sum mod 11 == 0 indicates valid.

    Known valid 9-digit: 123 456 782 (using standard test values is risky;
    actual production TFNs must pass this check)
    """
    clean = re.sub(r"[\s-]", "", value)
    if len(clean) == 9:
        weights = [1, 4, 3, 7, 5, 8, 6, 9, 10]
    elif len(clean) == 8:
        weights = [10, 7, 8, 4, 6, 3, 5, 1]
    else:
        return False
    if not clean.isdigit():
        return False
    total = sum(int(clean[i]) * weights[i] for i in range(len(clean)))
    return total % 11 == 0


def _validate_ca_sin(value: str) -> bool:
    """
    Canadian SIN: 9 digits, Luhn algorithm.
    SINs starting with 9 are for temporary residents (still valid format).
    Known invalid: 000-000-000.
    """
    from validators.luhn import Luhn
    clean = re.sub(r"[\s-]", "", value)
    if not re.match(r"^\d{9}$", clean):
        return False
    if clean == "000000000":
        return False
    return Luhn.validate(clean)


def _validate_sg_nric(value: str) -> bool:
    """
    Singapore NRIC / FIN weighted mod-11 checksum.
    Format: [S/T/F/G/M][7 digits][check letter]

    Weight array: 2, 7, 6, 5, 4, 3, 2 for positions 1-7.
    S/T prefix uses check letter set: JZIHGFEDCBA
    F/G prefix uses check letter set: XWUTRQPNMLK
    M prefix uses check letter set: XWUTRQPNMLK (same as F/G)

    Known valid: S1234567D
    """
    clean = re.sub(r"[\s-]", "", value.upper())
    if not re.match(r"^[STFGM]\d{7}[A-Z]$", clean):
        return False
    prefix = clean[0]
    digits = [int(c) for c in clean[1:8]]
    check_char = clean[8]
    weights = [2, 7, 6, 5, 4, 3, 2]
    total = sum(digits[i] * weights[i] for i in range(7))
    if prefix in ('T', 'G'):
        total += 4
    elif prefix == 'M':
        total += 3
    remainder = total % 11
    if prefix in ('S', 'T'):
        check_letters = "JZIHGFEDCBA"
    else:  # F, G, M
        check_letters = "XWUTRQPNMLK"
    expected = check_letters[remainder]
    return check_char == expected


def _validate_my_nric(value: str) -> bool:
    """
    Malaysia MyKad NRIC: YYMMDD-SS-XXXX (12 digits).
    Validates the date portion for plausibility.

    Known valid: 850101-14-1234
    """
    clean = re.sub(r"[\s-]", "", value)
    if not re.match(r"^\d{12}$", clean):
        return False
    month = int(clean[2:4])
    day = int(clean[4:6])
    if month < 1 or month > 12:
        return False
    if day < 1 or day > 31:
        return False
    return True


# --------------------------------------------------------------------------- #
# Pattern definitions                                                          #
# --------------------------------------------------------------------------- #

US_SSN = PiiPattern(
    id="US_SSN",
    name="US Social Security Number (SSN)",
    category=PatternCategory.GLOBAL_PII,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # Excludes 000-XX-XXXX, 666-XX-XXXX, 9XX-XX-XXXX
        r"\b(?!000|666|9\d{2})\d{3}[\s-]?(?!00)\d{2}[\s-]?(?!0{4})\d{4}\b",
    ],
    confidence_base=0.85,
    sensitivity="critical",
    context_keywords=[
        "ssn", "social_security", "tin", "social_security_number",
        "tax_id", "us_tax",
    ],
    validator=_validate_us_ssn,
    description="US Social Security Number (9 digits) issued by SSA; excludes ITINs.",
    false_positive_risk="medium",
    example="123-45-6789",
    country="US",
    is_locked=False,
)

US_EIN = PiiPattern(
    id="US_EIN",
    name="US Employer Identification Number (EIN)",
    category=PatternCategory.GLOBAL_PII,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        # Valid EIN prefix codes (IRS assigned)
        r"\b(?:0[1-6]|1[0-6]|2[0-7]|3[0-9]|4[0-8]|[5-9][0-9])\d{7}\b",
        # With hyphen
        r"\b(?:0[1-6]|1[0-6]|2[0-7]|3[0-9]|4[0-8]|[5-9][0-9])[\s-]\d{7}\b",
    ],
    confidence_base=0.75,
    sensitivity="medium",
    context_keywords=[
        "ein", "employer_id", "tax_id", "fein",
        "employer_identification", "business_tax_id",
    ],
    validator=None,
    description="9-digit US Employer Identification Number issued by IRS.",
    false_positive_risk="medium",
    example="12-3456789",
    country="US",
    is_locked=False,
)

US_PASSPORT = PiiPattern(
    id="US_PASSPORT",
    name="US Passport Number",
    category=PatternCategory.GLOBAL_PII,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # 9 alphanumeric — very broad, requires strong context
        r"(?i)\b[A-Z0-9]{9}\b",
    ],
    confidence_base=0.4,
    sensitivity="critical",
    context_keywords=[
        "us_passport", "american_passport", "us_travel_doc",
        "passport_number", "us_pp_no",
    ],
    validator=None,
    description="US passport number (9 alphanumeric); very broad pattern — requires strong column context.",
    false_positive_risk="critical",
    example="A12345678",
    country="US",
    is_locked=False,
)

UK_NHS = PiiPattern(
    id="UK_NHS",
    name="UK National Health Service Number",
    category=PatternCategory.GLOBAL_PII,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # 10 digits with optional spaces/hyphens between groups of 3-3-4
        r"\b\d{3}[\s-]?\d{3}[\s-]?\d{4}\b",
    ],
    confidence_base=0.9,
    sensitivity="critical",
    context_keywords=[
        "nhs", "nhs_number", "patient_id", "nhs_no",
        "national_health", "nhs_id",
    ],
    validator=_validate_uk_nhs,
    description="10-digit NHS number with weighted mod-11 checksum.",
    false_positive_risk="low",
    example="943-476-5919",
    country="GB",
    is_locked=False,
)

UK_NI = PiiPattern(
    id="UK_NI",
    name="UK National Insurance Number (NINO)",
    category=PatternCategory.GLOBAL_PII,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # AB123456C — excludes invalid prefixes BG,GB,KN,NK,NT,TN,ZZ
        r"(?i)\b(?!BG|GB|KN|NK|NT|TN|ZZ)[A-CEGHJ-PR-TW-Z][A-CEGHJ-NPR-TW-Z][0-9]{6}[A-D]\b",
    ],
    confidence_base=0.9,
    sensitivity="critical",
    context_keywords=[
        "ni_number", "national_insurance", "nino", "ni_no",
        "nat_insurance",
    ],
    validator=None,
    description="9-character UK National Insurance number with restricted letter prefix.",
    false_positive_risk="low",
    example="AB123456C",
    country="GB",
    is_locked=False,
)

UK_UTR = PiiPattern(
    id="UK_UTR",
    name="UK Unique Taxpayer Reference (UTR)",
    category=PatternCategory.GLOBAL_PII,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        # 10-digit number — needs strong context
        r"(?<!\d)\d{10}(?!\d)",
    ],
    confidence_base=0.6,
    sensitivity="high",
    context_keywords=[
        "utr", "taxpayer_reference", "self_assessment",
        "unique_taxpayer", "utr_number",
    ],
    validator=None,
    description="10-digit UK Unique Taxpayer Reference for self-assessment and corporation tax.",
    false_positive_risk="high",
    example="1234567890",
    country="GB",
    is_locked=False,
)

AU_TFN = PiiPattern(
    id="AU_TFN",
    name="Australian Tax File Number (TFN)",
    category=PatternCategory.GLOBAL_PII,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # 8 or 9 digits with optional spaces/hyphens
        r"\b\d{3}[\s-]?\d{3}[\s-]?\d{2,3}\b",
    ],
    confidence_base=0.85,
    sensitivity="critical",
    context_keywords=[
        "tfn", "tax_file_number", "australian_tax", "ato",
        "tax_file", "tfn_number",
    ],
    validator=_validate_au_tfn,
    description="8 or 9-digit Australian Tax File Number with weighted mod-11 checksum.",
    false_positive_risk="medium",
    example="123 456 782",
    country="AU",
    is_locked=False,
)

CA_SIN = PiiPattern(
    id="CA_SIN",
    name="Canadian Social Insurance Number (SIN)",
    category=PatternCategory.GLOBAL_PII,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # 9 digits, optional spaces/hyphens between groups of 3
        r"\b\d{3}[\s-]?\d{3}[\s-]?\d{3}\b",
    ],
    confidence_base=0.85,
    sensitivity="critical",
    context_keywords=[
        "sin", "social_insurance", "canada_tax", "canadian_sin",
        "sin_number",
    ],
    validator=_validate_ca_sin,
    description="9-digit Canadian Social Insurance Number validated with Luhn algorithm.",
    false_positive_risk="medium",
    example="046-454-286",
    country="CA",
    is_locked=False,
)

SG_NRIC = PiiPattern(
    id="SG_NRIC",
    name="Singapore NRIC / FIN",
    category=PatternCategory.GLOBAL_PII,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # [S/T/F/G/M] + 7 digits + 1 check letter
        r"(?i)\b[STFGM][0-9]{7}[A-Z]\b",
    ],
    confidence_base=0.9,
    sensitivity="critical",
    context_keywords=[
        "nric", "fin", "singapore_id", "ic_number",
        "sg_nric", "singpass",
    ],
    validator=_validate_sg_nric,
    description="9-character Singapore National Registration Identity Card / Foreign Identification Number.",
    false_positive_risk="low",
    example="S1234567D",
    country="SG",
    is_locked=False,
)

MY_NRIC = PiiPattern(
    id="MY_NRIC",
    name="Malaysia MyKad / NRIC",
    category=PatternCategory.GLOBAL_PII,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # YYMMDD-SS-XXXX (12 digits, hyphens optional)
        r"\b[0-9]{6}[\s-]?[0-9]{2}[\s-]?[0-9]{4}\b",
    ],
    confidence_base=0.75,
    sensitivity="critical",
    context_keywords=[
        "ic", "mykad", "nric", "malaysia_id", "my_ic",
        "mykad_number",
    ],
    validator=_validate_my_nric,
    description="12-digit Malaysian NRIC (MyKad) with embedded birth date (YYMMDD) and state code.",
    false_positive_risk="medium",
    example="850101-14-1234",
    country="MY",
    is_locked=False,
)

# European VAT numbers — separate entry per major country
EU_VAT_DE = PiiPattern(
    id="EU_VAT_DE",
    name="German VAT Number (Umsatzsteuer-ID)",
    category=PatternCategory.GLOBAL_PII,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        r"\bDE[0-9]{9}\b",
    ],
    confidence_base=0.85,
    sensitivity="medium",
    context_keywords=["vat", "ust_id", "steuer", "vat_number", "tax_id"],
    validator=None,
    description="German VAT number: DE followed by 9 digits.",
    false_positive_risk="low",
    example="DE123456789",
    country="DE",
    is_locked=False,
)

EU_VAT_FR = PiiPattern(
    id="EU_VAT_FR",
    name="French VAT Number (TVA)",
    category=PatternCategory.GLOBAL_PII,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        r"\bFR[A-HJ-NP-Z0-9]{2}[0-9]{9}\b",
    ],
    confidence_base=0.85,
    sensitivity="medium",
    context_keywords=["vat", "tva", "siren", "siret", "vat_number"],
    validator=None,
    description="French VAT number: FR + 2 alphanumeric chars + 9 digits.",
    false_positive_risk="low",
    example="FR12345678901",
    country="FR",
    is_locked=False,
)

EU_VAT_GB = PiiPattern(
    id="EU_VAT_GB",
    name="UK VAT Number",
    category=PatternCategory.GLOBAL_PII,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        r"\bGB[0-9]{9}(?:[0-9]{3})?\b",
    ],
    confidence_base=0.85,
    sensitivity="medium",
    context_keywords=["vat", "vat_number", "uk_vat", "hmrc"],
    validator=None,
    description="UK VAT number: GB followed by 9 or 12 digits.",
    false_positive_risk="low",
    example="GB123456789",
    country="GB",
    is_locked=False,
)

EU_VAT_IT = PiiPattern(
    id="EU_VAT_IT",
    name="Italian VAT Number (Partita IVA)",
    category=PatternCategory.GLOBAL_PII,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        r"\bIT[0-9]{11}\b",
    ],
    confidence_base=0.85,
    sensitivity="medium",
    context_keywords=["vat", "iva", "partita_iva", "vat_number"],
    validator=None,
    description="Italian VAT number: IT followed by 11 digits.",
    false_positive_risk="low",
    example="IT12345678901",
    country="IT",
    is_locked=False,
)

EU_VAT_ES = PiiPattern(
    id="EU_VAT_ES",
    name="Spanish VAT Number (NIF/CIF)",
    category=PatternCategory.GLOBAL_PII,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        r"\bES[A-Z0-9][0-9]{7}[A-Z0-9]\b",
    ],
    confidence_base=0.85,
    sensitivity="medium",
    context_keywords=["vat", "nif", "cif", "vat_number", "spain_tax"],
    validator=None,
    description="Spanish VAT number: ES + 1 alphanumeric + 7 digits + 1 alphanumeric.",
    false_positive_risk="low",
    example="ESA12345678",
    country="ES",
    is_locked=False,
)


GLOBAL_PII_PATTERNS = [
    US_SSN,
    US_EIN,
    US_PASSPORT,
    UK_NHS,
    UK_NI,
    UK_UTR,
    AU_TFN,
    CA_SIN,
    SG_NRIC,
    MY_NRIC,
    EU_VAT_DE,
    EU_VAT_FR,
    EU_VAT_GB,
    EU_VAT_IT,
    EU_VAT_ES,
]
