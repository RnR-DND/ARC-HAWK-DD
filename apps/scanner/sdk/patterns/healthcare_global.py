"""
Global Healthcare Patterns
============================
Medical coding and provider identification numbers.

Covered:
  ICD10_CODE     ICD-10 diagnosis code
  NPI_NUMBER     US National Provider Identifier (Luhn variant)
  NDC_DRUG_CODE  National Drug Code (US)
"""

import re
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from .base import PiiPattern, PatternCategory, DPDPASchedule


# --------------------------------------------------------------------------- #
# Validators                                                                   #
# --------------------------------------------------------------------------- #

def _validate_npi(value: str) -> bool:
    """
    US NPI: 10 digits, validated with Luhn algorithm using prefix 80840.
    The full number for Luhn check is: 80840 + npi[0:9] + npi[9] (check digit).

    Algorithm:
    1. Prepend "80840" to the 10-digit NPI.
    2. Run standard Luhn on the resulting 15-digit string.

    Known valid NPI: 1234567893 (per CMS test data)
    """
    clean = re.sub(r"[\s-]", "", value)
    if not re.match(r"^\d{10}$", clean):
        return False
    # Construct full number: 80840 + NPI
    full = "80840" + clean
    # Luhn validation
    total = 0
    parity = len(full) % 2
    for i, digit in enumerate(full):
        d = int(digit)
        if i % 2 == parity:
            d *= 2
            if d > 9:
                d -= 9
        total += d
    return total % 10 == 0


def _validate_icd10(value: str) -> bool:
    """
    ICD-10 code format validation.
    Format: [A-Z][0-9]{2}[.][0-9A-Z]{1,4}  (with optional dot after 3rd char)
    Category codes (3-char) are also valid: [A-Z][0-9]{2}
    """
    clean = value.strip().upper()
    # 3-char category code
    if re.match(r"^[A-Z][0-9]{2}$", clean):
        return True
    # Full code with optional dot
    if re.match(r"^[A-Z][0-9]{2}\.?[0-9A-Z]{1,4}$", clean):
        return True
    return False


def _validate_ndc(value: str) -> bool:
    """
    NDC format validation: 5-4-2, 4-4-2, or 5-3-2 segment layout.
    After stripping hyphens/spaces, total digits should be 10 or 11.
    """
    clean = re.sub(r"[\s-]", "", value)
    return re.match(r"^\d{10,11}$", clean) is not None


# --------------------------------------------------------------------------- #
# Pattern definitions                                                          #
# --------------------------------------------------------------------------- #

ICD10_CODE = PiiPattern(
    id="ICD10_CODE",
    name="ICD-10 Diagnosis Code",
    category=PatternCategory.HEALTHCARE_GLOBAL,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # Category code: letter + 2 digits (e.g. A09)
        r"(?i)\b[A-Z][0-9]{2}(?:\.[0-9A-Z]{1,4})?\b",
    ],
    confidence_base=0.75,
    sensitivity="high",
    context_keywords=[
        "icd", "icd10", "icd_10", "diagnosis", "condition_code",
        "dx_code", "diagnosis_code", "medical_code",
    ],
    validator=_validate_icd10,
    description="ICD-10-CM diagnosis code: letter + 2 digits + optional decimal subclassification.",
    false_positive_risk="medium",
    example="J18.9",
    country="GLOBAL",
    is_locked=False,
)

NPI_NUMBER = PiiPattern(
    id="NPI_NUMBER",
    name="US National Provider Identifier (NPI)",
    category=PatternCategory.HEALTHCARE_GLOBAL,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # 10-digit number; Luhn validation gates false positives
        r"(?<!\d)[0-9]{10}(?!\d)",
    ],
    confidence_base=0.85,
    sensitivity="high",
    context_keywords=[
        "npi", "provider_id", "national_provider", "npi_number",
        "provider_npi", "billing_npi",
    ],
    validator=_validate_npi,
    description="10-digit US National Provider Identifier with Luhn checksum (prefix 80840).",
    false_positive_risk="medium",
    example="1234567893",
    country="US",
    is_locked=False,
)

NDC_DRUG_CODE = PiiPattern(
    id="NDC_DRUG_CODE",
    name="National Drug Code (NDC)",
    category=PatternCategory.HEALTHCARE_GLOBAL,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # 5-4-2 format (most common)
        r"\b[0-9]{4,5}[\s\-][0-9]{3,4}[\s\-][0-9]{1,2}\b",
        # 11-digit compact (no separators)
        r"(?<!\d)[0-9]{11}(?!\d)",
    ],
    confidence_base=0.8,
    sensitivity="high",
    context_keywords=[
        "ndc", "drug_code", "medication_code", "formulary",
        "ndc_number", "drug_ndc", "rx_code",
    ],
    validator=_validate_ndc,
    description="National Drug Code (NDC): 10 or 11-digit identifier for US drug products.",
    false_positive_risk="medium",
    example="12345-6789-01",
    country="US",
    is_locked=False,
)


HEALTHCARE_GLOBAL_PATTERNS = [
    ICD10_CODE,
    NPI_NUMBER,
    NDC_DRUG_CODE,
]
