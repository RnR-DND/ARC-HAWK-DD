"""
PII Type Mapping — Maps user-selected PII types to regex patterns and Presidio entities.

This is the single source of truth for mapping between:
- Frontend PII type names (what users select)
- Fingerprint.yml pattern names (regex patterns)
- Presidio entity types (NLP recognizers)
"""

# Maps user-facing PII type → fingerprint.yml pattern name(s)
# Users select short names on frontend; these map to one or more regex patterns
PII_TO_FINGERPRINT_PATTERNS = {
    "AADHAAR": ["Aadhar"],
    "PAN": ["PAN"],
    "EMAIL": ["Email"],
    "PHONE": ["Phone_India", "Phone_International", "Phone_US"],
    "CREDIT_CARD": ["Credit_Card_Visa", "Credit_Card_MC", "Credit_Card_Amex", "Credit_Card_Discover"],
    "PASSPORT": ["Passport_India"],
    "VOTER_ID": ["Voter_ID"],
    "DRIVING_LICENSE": ["Drivers_License_India", "Drivers_License_US"],
    "UPI": ["UPI_ID"],
    "UPI_ID": ["UPI_ID"],  # alias — frontend sends UPI_ID
    "IFSC": ["IFSC_Code"],
    "BANK_ACCOUNT": ["Account_Number"],
    "GST": [],  # GST uses Presidio recognizer only, no fingerprint pattern
    # Contextual PII types (regex-assisted)
    "NAME": ["Full_Name"],
    "DOB": ["DOB"],
    "ADDRESS": ["Street_Address", "ZIP_Code", "PIN_Code_India"],
    "SSN": ["SSN"],
    "AGE": ["Age"],
    "GENDER": ["Gender"],
    # Secrets / credentials
    "AWS_KEY": ["AWS_Access_Key", "AWS_Secret_Key", "AWS_Session_Token", "AWS_S3_Bucket"],
    "GOOGLE_KEY": ["Google_API_Key", "Google_Service_Account", "Google_OAuth"],
    "AZURE_KEY": ["Azure_Storage_Key", "Azure_Client_Secret"],
    "SLACK_TOKEN": ["Slack_Token", "Slack_User_Token", "Slack_Webhook"],
    "API_KEY": ["API_Key_Generic", "Bearer_Token", "JWT_Token"],
    "PRIVATE_KEY": ["Private_Key_Header"],
    "DATABASE_URL": ["MongoDB_Connection", "PostgreSQL_Connection", "MySQL_Connection", "Redis_Connection"],
    "PAYMENT_KEY": ["Stripe_API_Key", "PayPal_Braintree_Token", "Square_Access_Token"],
    "IP_ADDRESS": ["IP_Address"],
    "MAC_ADDRESS": ["MAC_Address"],
    "CVV": ["CVV"],
}

# Maps user-facing PII type → Presidio entity type(s)
# These are the entity names registered in sdk/engine.py recognizers
PII_TO_PRESIDIO_ENTITIES = {
    "AADHAAR": ["IN_AADHAAR"],
    "PAN": ["IN_PAN"],
    "EMAIL": ["EMAIL_ADDRESS"],
    "PHONE": ["IN_PHONE"],
    "CREDIT_CARD": ["CREDIT_CARD"],
    "PASSPORT": ["IN_PASSPORT"],
    "VOTER_ID": ["IN_VOTER_ID"],
    "DRIVING_LICENSE": ["IN_DRIVING_LICENSE"],
    "UPI": ["IN_UPI"],
    "UPI_ID": ["IN_UPI"],  # alias
    "IFSC": ["IN_IFSC"],
    "BANK_ACCOUNT": ["IN_BANK_ACCOUNT"],
    "GST": ["IN_GST"],
}

# Reverse lookup: fingerprint pattern name → user PII type
_PATTERN_TO_PII_TYPE = {}
for pii_type, patterns in PII_TO_FINGERPRINT_PATTERNS.items():
    for pattern in patterns:
        _PATTERN_TO_PII_TYPE[pattern.lower()] = pii_type


def get_fingerprint_patterns_for_pii_types(selected_pii_types: list[str]) -> set[str]:
    """
    Given user-selected PII types, return the set of fingerprint.yml pattern names to run.

    Args:
        selected_pii_types: e.g. ["PAN", "AADHAAR", "EMAIL"]

    Returns:
        Set of pattern names from fingerprint.yml, e.g. {"PAN", "Aadhar", "Email"}
        Empty set means run ALL patterns (no filtering).
    """
    if not selected_pii_types:
        return set()  # empty = no filter = run all

    patterns = set()
    for pii_type in selected_pii_types:
        pii_upper = pii_type.upper()
        if pii_upper in PII_TO_FINGERPRINT_PATTERNS:
            patterns.update(PII_TO_FINGERPRINT_PATTERNS[pii_upper])
    return patterns


def get_presidio_entities_for_pii_types(selected_pii_types: list[str]) -> list[str] | None:
    """
    Given user-selected PII types, return the Presidio entity names to scan for.

    Args:
        selected_pii_types: e.g. ["PAN", "AADHAAR", "EMAIL"]

    Returns:
        List of Presidio entity type strings, or None (= scan all entities).
    """
    if not selected_pii_types:
        return None  # None = run all entities

    entities = set()
    for pii_type in selected_pii_types:
        pii_upper = pii_type.upper()
        if pii_upper in PII_TO_PRESIDIO_ENTITIES:
            entities.update(PII_TO_PRESIDIO_ENTITIES[pii_upper])

    return list(entities) if entities else None


def filter_fingerprint_patterns(all_patterns: dict, selected_pii_types: list[str]) -> dict:
    """
    Filter a fingerprint pattern dict to only include patterns for selected PII types.

    Args:
        all_patterns: Full fingerprint.yml dict {"Email": "regex...", "PAN": "regex...", ...}
        selected_pii_types: User-selected PII types, e.g. ["PAN", "EMAIL"]

    Returns:
        Filtered dict with only relevant patterns. Returns all if selected_pii_types is empty.
    """
    if not selected_pii_types:
        return all_patterns  # no filter

    allowed_patterns = get_fingerprint_patterns_for_pii_types(selected_pii_types)
    if not allowed_patterns:
        return all_patterns  # no mapping found, run all as fallback

    # Case-insensitive match
    allowed_lower = {p.lower() for p in allowed_patterns}
    return {
        name: regex
        for name, regex in all_patterns.items()
        if name.lower() in allowed_lower
    }
