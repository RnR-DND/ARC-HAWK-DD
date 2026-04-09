"""
India Corporate Patterns
=========================
PII patterns for Indian corporate/business identifiers.

Covered:
  IN_CIN           Company Identification Number
  IN_LLPIN         LLP Identification Number
  IN_MSME_UDYAM    Udyam / MSME Registration Number
  IN_FSSAI         Food Safety & Standards Authority License
  IN_DRUG_LICENSE  Drug / Pharmacy Manufacturing License
"""

import re
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from patterns.base import PiiPattern, PatternCategory, DPDPASchedule


# --------------------------------------------------------------------------- #
# Validators                                                                   #
# --------------------------------------------------------------------------- #

def _validate_cin(value: str) -> bool:
    """
    CIN format: [LU][5-digit NIC][2-letter state][4-digit year][3-letter type][6 digits]
    Total 21 characters.
    Example: L17110MH1973PLC019786
    """
    clean = re.sub(r"\s", "", value.upper())
    if len(clean) != 21:
        return False
    pattern = re.compile(
        r"^[LU][0-9]{5}[A-Z]{2}[0-9]{4}[A-Z]{3}[0-9]{6}$"
    )
    if not pattern.match(clean):
        return False
    # Validate company type
    valid_types = {
        "PLC", "PTC", "LTD", "LLC", "OPC", "NPL", "FLC", "FTC",
        "GAP", "GOI", "GAT", "GAL", "GAD",
    }
    company_type = clean[15:18]
    return company_type in valid_types


def _validate_udyam(value: str) -> bool:
    """
    Udyam: UDYAM-<2-letter state>-<2 digits>-<7 digits>
    Example: UDYAM-MH-01-0001234
    """
    clean = re.sub(r"[\s-]", "", value.upper())
    return bool(re.match(r"^UDYAM[A-Z]{2}[0-9]{2}[0-9]{7}$", clean))


# --------------------------------------------------------------------------- #
# Pattern definitions                                                          #
# --------------------------------------------------------------------------- #

IN_CIN = PiiPattern(
    id="IN_CIN",
    name="Company Identification Number (CIN)",
    category=PatternCategory.INDIA_CORPORATE,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        # L/U + 5 NIC digits + 2-alpha state + 4-digit year + 3-alpha type + 6 digits
        r"(?i)\b[LUlu][0-9]{5}[A-Z]{2}[0-9]{4}[A-Z]{3}[0-9]{6}\b",
    ],
    confidence_base=0.9,
    sensitivity="medium",
    context_keywords=[
        "cin", "company_id", "mca", "corporate_identity",
        "company_identification", "cin_number",
    ],
    validator=_validate_cin,
    description="21-character Company Identification Number issued by MCA for registered companies.",
    false_positive_risk="low",
    example="L17110MH1973PLC019786",
    country="IN",
    is_locked=False,
)

IN_LLPIN = PiiPattern(
    id="IN_LLPIN",
    name="LLP Identification Number (LLPIN)",
    category=PatternCategory.INDIA_CORPORATE,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        # 3 uppercase alpha + 4 digits (e.g. AAB-1234)
        r"(?i)\b[A-Z]{3}[\s-]?[0-9]{4}\b",
    ],
    confidence_base=0.7,
    sensitivity="low",
    context_keywords=[
        "llpin", "llp", "limited_liability_partnership",
        "llp_id", "llp_number",
    ],
    validator=None,
    description="7-character identification number for Limited Liability Partnerships issued by MCA.",
    false_positive_risk="high",
    example="AAB-1234",
    country="IN",
    is_locked=False,
)

IN_MSME_UDYAM = PiiPattern(
    id="IN_MSME_UDYAM",
    name="Udyam / MSME Registration Number",
    category=PatternCategory.INDIA_CORPORATE,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        # UDYAM-<state>-<seq> with optional separators
        r"(?i)\bUDYAM[\s-]?[A-Z]{2}[\s-]?[0-9]{2}[\s-]?[0-9]{7}\b",
    ],
    confidence_base=0.95,
    sensitivity="low",
    context_keywords=[
        "udyam", "msme", "udyog_aadhaar", "msme_registration",
        "udyam_number", "udyam_certificate",
    ],
    validator=_validate_udyam,
    description="Udyam Registration Number (formerly Udyog Aadhaar) for MSME enterprises.",
    false_positive_risk="low",
    example="UDYAM-MH-01-0001234",
    country="IN",
    is_locked=False,
)

IN_FSSAI = PiiPattern(
    id="IN_FSSAI",
    name="FSSAI Food Safety License Number",
    category=PatternCategory.INDIA_CORPORATE,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        # 14-digit, first digit 1-9 (state code embedded)
        r"(?<!\d)[1-9][0-9]{13}(?!\d)",
    ],
    confidence_base=0.65,
    sensitivity="low",
    context_keywords=[
        "fssai", "food_license", "fbo_id", "food_safety",
        "food_business", "fssai_number",
    ],
    validator=None,
    description="14-digit FSSAI license/registration number for food business operators.",
    false_positive_risk="high",
    example="10016011002458",
    country="IN",
    is_locked=False,
)

IN_DRUG_LICENSE = PiiPattern(
    id="IN_DRUG_LICENSE",
    name="Drug / Pharmacy License Number",
    category=PatternCategory.INDIA_CORPORATE,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        # State/type/number: e.g. MH/MFG/012345 or KA/GEN/000123
        r"(?i)\b[A-Z]{2}[\s/\-]?[A-Z]{2,3}[\s/\-]?[0-9]{5,7}\b",
    ],
    confidence_base=0.6,
    sensitivity="low",
    context_keywords=[
        "drug_license", "pharmacy_license", "manufacturing_license",
        "dl_number", "drug_lic", "pharmaceutical_license",
    ],
    validator=None,
    description="State-issued drug manufacturing or pharmacy retail license number under Drugs & Cosmetics Act.",
    false_positive_risk="high",
    example="MH/MFG/012345",
    country="IN",
    is_locked=False,
)


INDIA_CORPORATE_PATTERNS = [
    IN_CIN,
    IN_LLPIN,
    IN_MSME_UDYAM,
    IN_FSSAI,
    IN_DRUG_LICENSE,
]
