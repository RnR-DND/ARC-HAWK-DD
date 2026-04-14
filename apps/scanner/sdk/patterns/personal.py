"""
Personal / Demographic PII Patterns
=====================================
General personal data: dates, network addresses, biometrics, etc.

Covered:
  DATE_OF_BIRTH      Date of birth in multiple formats
  GENDER_MARKER      Gender / biological sex column marker
  BLOOD_GROUP        Blood group / blood type
  IP_ADDRESS_V4      IPv4 address (excludes private/loopback)
  IP_ADDRESS_V6      IPv6 address
  MAC_ADDRESS        Ethernet MAC address
  GPS_COORDINATES    Latitude + longitude pair
  PERSON_NAME        Full name (NER-assisted, low base confidence)
  PHYSICAL_ADDRESS   Street address heuristic
"""

import re
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from .base import PiiPattern, PatternCategory, DPDPASchedule


# --------------------------------------------------------------------------- #
# Validators                                                                   #
# --------------------------------------------------------------------------- #

def _validate_ipv4(value: str) -> bool:
    """
    IPv4 address validation: each octet 0-255.
    Excludes private/loopback ranges to reduce FP rate.
    Allowed: public IPs only.
    """
    parts = value.strip().split(".")
    if len(parts) != 4:
        return False
    try:
        octets = [int(p) for p in parts]
    except ValueError:
        return False
    if not all(0 <= o <= 255 for o in octets):
        return False
    # Exclude loopback
    if octets[0] == 127:
        return False
    # Exclude private class A (10.x.x.x)
    if octets[0] == 10:
        return False
    # Exclude private class B (172.16-31.x.x)
    if octets[0] == 172 and 16 <= octets[1] <= 31:
        return False
    # Exclude private class C (192.168.x.x)
    if octets[0] == 192 and octets[1] == 168:
        return False
    return True


def _validate_dob(value: str) -> bool:
    """
    Validate date of birth for plausibility (year 1900-2025, valid month/day).
    """
    import datetime
    clean = value.strip()
    # Try YYYY-MM-DD or YYYY/MM/DD
    m = re.match(r"^(?P<y>(?:19|20)\d{2})[/.\-](?P<m>0[1-9]|1[0-2])[/.\-](?P<d>0[1-9]|[12]\d|3[01])$", clean)
    if m:
        y, mon, d = int(m["y"]), int(m["m"]), int(m["d"])
    else:
        # Try DD/MM/YYYY or DD-MM-YYYY
        m = re.match(r"^(?P<d>0[1-9]|[12]\d|3[01])[/.\-](?P<m>0[1-9]|1[0-2])[/.\-](?P<y>(?:19|20)\d{2})$", clean)
        if m:
            y, mon, d = int(m["y"]), int(m["m"]), int(m["d"])
        else:
            return False
    try:
        datetime.date(y, mon, d)
        return True
    except ValueError:
        return False


def _validate_mac(value: str) -> bool:
    """MAC address: exactly 6 groups of 2 hex digits separated by : or -."""
    clean = value.strip().upper()
    return bool(re.match(r"^(?:[0-9A-F]{2}[:\-]){5}[0-9A-F]{2}$", clean))


def _validate_gps(value: str) -> bool:
    """
    GPS coordinate pair: lat in [-90, 90], lon in [-180, 180].
    Input may be "lat, lon" or "lat lon".
    """
    m = re.match(
        r"^\s*(?P<lat>[-+]?(?:[1-8]?\d(?:\.\d+)?|90(?:\.0+)?))"
        r"[\s,]+"
        r"(?P<lon>[-+]?(?:180(?:\.0+)?|(?:1[0-7]\d|[1-9]?\d)(?:\.\d+)?))\s*$",
        value,
    )
    if not m:
        return False
    lat, lon = float(m["lat"]), float(m["lon"])
    return -90 <= lat <= 90 and -180 <= lon <= 180


# --------------------------------------------------------------------------- #
# Pattern definitions                                                          #
# --------------------------------------------------------------------------- #

DATE_OF_BIRTH = PiiPattern(
    id="DATE_OF_BIRTH",
    name="Date of Birth",
    category=PatternCategory.PERSONAL,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # DD/MM/YYYY or DD-MM-YYYY or DD.MM.YYYY
        r"\b(?:0[1-9]|[12][0-9]|3[01])[/.\-](?:0[1-9]|1[0-2])[/.\-](?:19|20)\d{2}\b",
        # YYYY-MM-DD or YYYY/MM/DD
        r"\b(?:19|20)\d{2}[/.\-](?:0[1-9]|1[0-2])[/.\-](?:0[1-9]|[12][0-9]|3[01])\b",
        # DD-Mon-YYYY (e.g. 01-Jan-1990)
        r"\b(?:0[1-9]|[12][0-9]|3[01])[\s\-]"
        r"(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[\s\-]"
        r"(?:19|20)\d{2}\b",
    ],
    confidence_base=0.7,
    sensitivity="high",
    context_keywords=[
        "dob", "date_of_birth", "birth_date", "birthdate",
        "born_on", "date_birth", "birth_dt",
    ],
    validator=_validate_dob,
    description="Date of birth in multiple common formats (DD/MM/YYYY, YYYY-MM-DD, DD-Mon-YYYY).",
    false_positive_risk="medium",
    example="15/08/1990",
    country="GLOBAL",
    is_locked=False,
)

GENDER_MARKER = PiiPattern(
    id="GENDER_MARKER",
    name="Gender / Biological Sex Marker",
    category=PatternCategory.PERSONAL,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # Common single-character values
        r"(?i)\b(?:Male|Female|M|F|Other|Non[\s-]?binary|Prefer not to say|Unknown)\b",
    ],
    confidence_base=0.8,
    sensitivity="high",
    context_keywords=[
        "gender", "sex", "biological_sex", "gender_identity",
        "sex_at_birth",
    ],
    validator=None,
    description="Gender or biological sex value in a column explicitly named gender/sex.",
    false_positive_risk="medium",
    example="Female",
    country="GLOBAL",
    is_locked=False,
)

BLOOD_GROUP = PiiPattern(
    id="BLOOD_GROUP",
    name="Blood Group / Type",
    category=PatternCategory.PERSONAL,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # ABO + Rh system
        r"(?i)\b(?:A|B|AB|O)[+\-]\b",
        # Spelled out
        r"(?i)\b(?:A|B|AB|O)\s*(?:positive|negative|pos|neg)\b",
    ],
    confidence_base=0.7,
    sensitivity="high",
    context_keywords=[
        "blood_group", "blood_type", "blood", "rh_factor",
        "abo_type",
    ],
    validator=None,
    description="ABO blood group and Rh factor (e.g. A+, O-, AB positive).",
    false_positive_risk="medium",
    example="O+",
    country="GLOBAL",
    is_locked=False,
)

IP_ADDRESS_V4 = PiiPattern(
    id="IP_ADDRESS_V4",
    name="IPv4 Address (Public)",
    category=PatternCategory.PERSONAL,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        # Full dotted-quad, each octet 0-255
        r"\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}"
        r"(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b",
    ],
    confidence_base=0.7,
    sensitivity="medium",
    context_keywords=[
        "ip", "ip_address", "client_ip", "remote_addr",
        "source_ip", "request_ip", "user_ip",
    ],
    validator=_validate_ipv4,
    description="Public IPv4 address (excludes 10.x, 172.16-31.x, 192.168.x, 127.x).",
    false_positive_risk="medium",
    example="203.0.113.42",
    country="GLOBAL",
    is_locked=False,
)

IP_ADDRESS_V6 = PiiPattern(
    id="IP_ADDRESS_V6",
    name="IPv6 Address",
    category=PatternCategory.PERSONAL,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        # Full 8-group IPv6
        r"\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b",
        # Compressed IPv6 with ::
        r"\b(?:[0-9a-fA-F]{1,4}:){1,7}:\b",
        r"\b::(?:[0-9a-fA-F]{1,4}:){0,6}[0-9a-fA-F]{1,4}\b",
    ],
    confidence_base=0.85,
    sensitivity="medium",
    context_keywords=[
        "ipv6", "ip6", "ip_address", "client_ip", "remote_addr",
        "source_ip",
    ],
    validator=None,
    description="IPv6 address in full or compressed notation.",
    false_positive_risk="low",
    example="2001:0db8:85a3:0000:0000:8a2e:0370:7334",
    country="GLOBAL",
    is_locked=False,
)

MAC_ADDRESS = PiiPattern(
    id="MAC_ADDRESS",
    name="MAC Address (Hardware)",
    category=PatternCategory.PERSONAL,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        # Colon-separated hex pairs
        r"\b(?:[0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}\b",
        # Hyphen-separated hex pairs
        r"\b(?:[0-9a-fA-F]{2}-){5}[0-9a-fA-F]{2}\b",
    ],
    confidence_base=0.85,
    sensitivity="medium",
    context_keywords=[
        "mac", "hardware_address", "physical_address", "mac_address",
        "ethernet", "device_mac",
    ],
    validator=_validate_mac,
    description="48-bit MAC address in colon- or hyphen-separated hex format.",
    false_positive_risk="low",
    example="00:1A:2B:3C:4D:5E",
    country="GLOBAL",
    is_locked=False,
)

GPS_COORDINATES = PiiPattern(
    id="GPS_COORDINATES",
    name="GPS Coordinates (Lat/Lon)",
    category=PatternCategory.PERSONAL,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # lat, lon pair with decimal points
        r"(?i)\b[-+]?(?:[1-8]?\d(?:\.\d+)?|90(?:\.0+)?),"
        r"\s*[-+]?(?:180(?:\.0+)?|(?:1[0-7]\d|[1-9]?\d)(?:\.\d+)?)\b",
    ],
    confidence_base=0.85,
    sensitivity="high",
    context_keywords=[
        "lat", "lon", "latitude", "longitude", "gps",
        "coordinates", "location", "geolocation", "geo",
    ],
    validator=_validate_gps,
    description="WGS-84 GPS coordinate pair: latitude [-90,90] and longitude [-180,180].",
    false_positive_risk="medium",
    example="28.6139, 77.2090",
    country="GLOBAL",
    is_locked=False,
)

PERSON_NAME = PiiPattern(
    id="PERSON_NAME",
    name="Person Name (NER-assisted)",
    category=PatternCategory.PERSONAL,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        # Heuristic: 2-4 capitalised words, each 2-30 chars, separated by space
        r"\b[A-Z][a-z]{1,29}(?:\s+[A-Z][a-z]{1,29}){1,3}\b",
    ],
    confidence_base=0.5,
    sensitivity="medium",
    context_keywords=[
        "name", "full_name", "first_name", "last_name", "customer_name",
        "employee_name", "patient_name", "person_name",
    ],
    validator=None,
    description=(
        "Person name detected via capitalized-word heuristic. "
        "NER (spaCy PERSON entity) should supplement for higher confidence."
    ),
    false_positive_risk="high",
    example="Rahul Kumar Sharma",
    country="GLOBAL",
    is_locked=False,
)

PHYSICAL_ADDRESS = PiiPattern(
    id="PHYSICAL_ADDRESS",
    name="Physical Street Address",
    category=PatternCategory.PERSONAL,
    dpdpa_schedule=DPDPASchedule.PERSONAL_DATA,
    patterns=[
        # India: number + street name + city/area indicator
        r"(?i)\b\d{1,4}[\s,]+[A-Za-z0-9\s,.\-]+?"
        r"(?:Road|Street|Lane|Nagar|Colony|Sector|Block|Phase|Avenue|Marg|Path|Layout)"
        r"[A-Za-z0-9\s,.\-]{0,60}\b",
    ],
    confidence_base=0.6,
    sensitivity="medium",
    context_keywords=[
        "address", "addr", "street", "city", "pincode",
        "postal_address", "mailing_address", "residence",
    ],
    validator=None,
    description=(
        "Indian street address heuristic: number + named road/lane/nagar/colony type. "
        "High false-positive risk — use only with strong column context."
    ),
    false_positive_risk="high",
    example="42, MG Road, Andheri East",
    country="IN",
    is_locked=False,
)


PERSONAL_PATTERNS = [
    DATE_OF_BIRTH,
    GENDER_MARKER,
    BLOOD_GROUP,
    IP_ADDRESS_V4,
    IP_ADDRESS_V6,
    MAC_ADDRESS,
    GPS_COORDINATES,
    PERSON_NAME,
    PHYSICAL_ADDRESS,
]
