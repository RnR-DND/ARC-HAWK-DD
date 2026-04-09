"""
India Healthcare Patterns
==========================
PII patterns for Indian health identifiers.

Covered:
  IN_ABHA_HEALTH_ID   Ayushman Bharat Health Account (ABHA) Number
                      (also in india_identity; here as healthcare category alias)
  IN_MCI_REG          Medical Council of India / NMC Registration Number
  IN_PHARMACY_REG     State Pharmacy Council Registration Number
"""

import re
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from patterns.base import PiiPattern, PatternCategory, DPDPASchedule


# --------------------------------------------------------------------------- #
# Pattern definitions                                                          #
# --------------------------------------------------------------------------- #

IN_MCI_REG = PiiPattern(
    id="IN_MCI_REG",
    name="Medical Council of India / NMC Registration Number",
    category=PatternCategory.INDIA_HEALTHCARE,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        # State MCI registrations: typically 5-6 digit numeric
        r"(?<!\d)[0-9]{5,6}(?!\d)",
        # NMC national registration: alpha-prefix + digits, e.g. UK/123456
        r"(?i)\b[A-Z]{2,3}[/\-]?[0-9]{5,7}\b",
    ],
    confidence_base=0.55,
    sensitivity="medium",
    context_keywords=[
        "mci_reg", "doctor_reg", "nmc_number", "medical_registration",
        "doctor_id", "physician_id", "mci",
    ],
    validator=None,
    description="Medical Council of India / NMC registration number issued to licensed doctors.",
    false_positive_risk="high",
    example="123456",
    country="IN",
    is_locked=False,
)

IN_PHARMACY_REG = PiiPattern(
    id="IN_PHARMACY_REG",
    name="State Pharmacy Council Registration Number",
    category=PatternCategory.INDIA_HEALTHCARE,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        # Typically state code + sequence: e.g. MH/001234 or TNPC/12345
        r"(?i)\b[A-Z]{2,5}[/\-]?[0-9]{4,7}\b",
    ],
    confidence_base=0.5,
    sensitivity="medium",
    context_keywords=[
        "pharmacy_reg", "pharmacist_reg", "pharmacy_council",
        "pharmacist_id", "pharmacy_number",
    ],
    validator=None,
    description="State pharmacy council registration number for licensed pharmacists.",
    false_positive_risk="high",
    example="MH/001234",
    country="IN",
    is_locked=False,
)


INDIA_HEALTHCARE_PATTERNS = [
    IN_MCI_REG,
    IN_PHARMACY_REG,
]
