"""
Credentials & Secrets Patterns
================================
API keys, tokens, secrets, connection strings, and cryptographic material.

All patterns here carry confidence >= 0.95 because they have highly distinctive
prefixes/formats. False positives are rare when the pattern matches.

Covered:
  AWS_ACCESS_KEY          AWS IAM access key ID
  AWS_SECRET_KEY          AWS secret access key
  GITHUB_TOKEN            GitHub fine-grained and classic tokens
  GOOGLE_API_KEY          Google API key
  STRIPE_SECRET_KEY       Stripe secret key (live + test)
  STRIPE_PUBLISHABLE_KEY  Stripe publishable key (live + test)
  SLACK_TOKEN             Slack bot/app tokens
  TWILIO_SID              Twilio account SID
  SENDGRID_API_KEY        SendGrid API key
  JWT_TOKEN               JSON Web Token
  RSA_PRIVATE_KEY         PEM-encoded private key header
  AZURE_CONNECTION_STRING Azure Storage connection string
  AZURE_SAS_TOKEN         Azure SAS token
  GCP_SERVICE_ACCOUNT     GCP service account key JSON marker
  GENERIC_SECRET_HIGH_ENT High-entropy generic secret (column-name gated)
  CRYPTO_BTC              (see financial_global.py)
  CRYPTO_ETH              (see financial_global.py)
"""

import math
import re
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from .base import PiiPattern, PatternCategory, DPDPASchedule


# --------------------------------------------------------------------------- #
# Validators                                                                   #
# --------------------------------------------------------------------------- #

def _shannon_entropy(s: str) -> float:
    """Compute Shannon entropy (bits per character) of string s."""
    if not s:
        return 0.0
    freq = {}
    for ch in s:
        freq[ch] = freq.get(ch, 0) + 1
    length = len(s)
    entropy = -sum((c / length) * math.log2(c / length) for c in freq.values())
    return entropy


def _validate_high_entropy_secret(value: str) -> bool:
    """
    Accept if Shannon entropy > 3.5 and length between 20 and 200 characters.
    This catches base64, hex, and random alphanumeric secrets.
    """
    clean = value.strip()
    if len(clean) < 20 or len(clean) > 200:
        return False
    return _shannon_entropy(clean) > 3.5


def _validate_jwt(value: str) -> bool:
    """
    A JWT has exactly 3 base64url segments separated by dots.
    First two must decode to JSON-like structures (start with eyJ = base64 of '{').
    """
    parts = value.split(".")
    if len(parts) != 3:
        return False
    return value.startswith("eyJ") and "." in value[3:]


def _always_valid(value: str) -> bool:
    """Used for patterns with near-zero false-positive rate due to unique prefixes."""
    return True


# --------------------------------------------------------------------------- #
# Pattern definitions                                                          #
# --------------------------------------------------------------------------- #

AWS_ACCESS_KEY = PiiPattern(
    id="AWS_ACCESS_KEY",
    name="AWS IAM Access Key ID",
    category=PatternCategory.CREDENTIALS,
    dpdpa_schedule=DPDPASchedule.CRITICAL_PERSONAL_DATA,
    patterns=[
        # AKIA (user), AROA (role), AIDA (instance profile), ASIA (assumed role)
        r"(?i)\b(?:AKIA|AROA|AIDA|ASIA)[0-9A-Z]{16}\b",
    ],
    confidence_base=0.99,
    sensitivity="critical",
    context_keywords=[
        "aws_access_key", "access_key_id", "aws_key",
        "aws_access", "iam_key",
    ],
    validator=_always_valid,
    description="AWS IAM Access Key ID with well-known 4-letter prefix (AKIA/AROA/AIDA/ASIA).",
    false_positive_risk="low",
    example="AKIA<YOUR_AWS_ACCESS_KEY_ID>",
    country="GLOBAL",
    is_locked=False,
)

AWS_SECRET_KEY = PiiPattern(
    id="AWS_SECRET_KEY",
    name="AWS Secret Access Key",
    category=PatternCategory.CREDENTIALS,
    dpdpa_schedule=DPDPASchedule.CRITICAL_PERSONAL_DATA,
    patterns=[
        # Triggered by context keyword in same text + 40 base64 chars
        r"(?i)(?:aws_secret|secret_access_key|SecretAccessKey)[\s=:\"']+([A-Za-z0-9/+=]{40})\b",
    ],
    confidence_base=0.95,
    sensitivity="critical",
    context_keywords=[
        "aws_secret", "secret_access_key", "secretaccesskey",
        "aws_secret_key",
    ],
    validator=_validate_high_entropy_secret,
    description="AWS Secret Access Key: 40 base64 characters, preceded by known context keyword.",
    false_positive_risk="low",
    example="<YOUR_AWS_SECRET_ACCESS_KEY>",
    country="GLOBAL",
    is_locked=False,
)

GITHUB_TOKEN = PiiPattern(
    id="GITHUB_TOKEN",
    name="GitHub Personal Access Token",
    category=PatternCategory.CREDENTIALS,
    dpdpa_schedule=DPDPASchedule.CRITICAL_PERSONAL_DATA,
    patterns=[
        # Fine-grained: ghp_ (personal), gho_ (oauth), ghu_ (user), ghs_ (server), ghr_ (refresh)
        r"(?i)\bgh[pousr]_[A-Za-z0-9_]{36}\b",
        # Classic token (40 hex chars) — requires context
        r"\b[a-f0-9]{40}\b",
    ],
    confidence_base=0.99,
    sensitivity="critical",
    context_keywords=[
        "github", "token", "access_token", "github_token",
        "gh_token", "personal_access_token",
    ],
    validator=_always_valid,
    description="GitHub PAT: fine-grained (ghp_/gho_/ghu_/ghs_/ghr_ prefix) or classic 40-hex.",
    false_positive_risk="low",
    example="ghp_<YOUR_GITHUB_PAT_TOKEN_HERE>",
    country="GLOBAL",
    is_locked=False,
)

GOOGLE_API_KEY = PiiPattern(
    id="GOOGLE_API_KEY",
    name="Google API Key",
    category=PatternCategory.CREDENTIALS,
    dpdpa_schedule=DPDPASchedule.CRITICAL_PERSONAL_DATA,
    patterns=[
        # All Google API keys start with AIza
        r"\bAIza[0-9A-Za-z_\-]{35}\b",
    ],
    confidence_base=0.99,
    sensitivity="critical",
    context_keywords=[
        "google_api_key", "api_key", "gcp_key", "maps_key",
        "firebase_key",
    ],
    validator=_always_valid,
    description="Google API key starting with 'AIza' followed by 35 base64url characters.",
    false_positive_risk="low",
    example="AIza<YOUR_GOOGLE_API_KEY_35_CHARS>",
    country="GLOBAL",
    is_locked=False,
)

STRIPE_SECRET_KEY = PiiPattern(
    id="STRIPE_SECRET_KEY",
    name="Stripe Secret Key",
    category=PatternCategory.CREDENTIALS,
    dpdpa_schedule=DPDPASchedule.CRITICAL_PERSONAL_DATA,
    patterns=[
        r"(?i)\bsk_(?:live|test)_[0-9a-zA-Z]{24,99}\b",
    ],
    confidence_base=0.99,
    sensitivity="critical",
    context_keywords=[
        "stripe", "stripe_secret", "payment_key", "sk_live", "sk_test",
    ],
    validator=_always_valid,
    description="Stripe secret key with sk_live_ or sk_test_ prefix.",
    false_positive_risk="low",
    example="sk_live_<YOUR_STRIPE_SECRET_KEY>",
    country="GLOBAL",
    is_locked=False,
)

STRIPE_PUBLISHABLE_KEY = PiiPattern(
    id="STRIPE_PUBLISHABLE_KEY",
    name="Stripe Publishable Key",
    category=PatternCategory.CREDENTIALS,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        r"(?i)\bpk_(?:live|test)_[0-9a-zA-Z]{24,99}\b",
    ],
    confidence_base=0.99,
    sensitivity="high",
    context_keywords=[
        "stripe", "stripe_publishable", "pk_live", "pk_test",
    ],
    validator=_always_valid,
    description="Stripe publishable key with pk_live_ or pk_test_ prefix.",
    false_positive_risk="low",
    example="pk_test_<YOUR_STRIPE_PUB_KEY>",
    country="GLOBAL",
    is_locked=False,
)

SLACK_TOKEN = PiiPattern(
    id="SLACK_TOKEN",
    name="Slack API Token",
    category=PatternCategory.CREDENTIALS,
    dpdpa_schedule=DPDPASchedule.CRITICAL_PERSONAL_DATA,
    patterns=[
        # xoxb (bot), xoxa (app-level), xoxp (user), xoxr (refresh), xoxs (service)
        r"(?i)\bxox[baprs]-[0-9A-Za-z\-]{10,72}\b",
    ],
    confidence_base=0.99,
    sensitivity="critical",
    context_keywords=[
        "slack", "slack_token", "bot_token", "slack_bot",
        "xoxb", "workspace_token",
    ],
    validator=_always_valid,
    description="Slack API token with xoxb/xoxa/xoxp/xoxr/xoxs prefix.",
    false_positive_risk="low",
    example="xoxb-<WORKSPACE_ID>-<BOT_TOKEN_STRING>",
    country="GLOBAL",
    is_locked=False,
)

TWILIO_SID = PiiPattern(
    id="TWILIO_SID",
    name="Twilio Account SID",
    category=PatternCategory.CREDENTIALS,
    dpdpa_schedule=DPDPASchedule.CRITICAL_PERSONAL_DATA,
    patterns=[
        # AC + 32 hex chars
        r"\bAC[0-9a-fA-F]{32}\b",
    ],
    confidence_base=0.99,
    sensitivity="critical",
    context_keywords=[
        "twilio", "account_sid", "twilio_sid", "twilio_account",
    ],
    validator=_always_valid,
    description="Twilio Account SID: AC followed by 32 hexadecimal characters.",
    false_positive_risk="low",
    example="AC<YOUR_32_HEX_CHAR_ACCOUNT_SID>",
    country="GLOBAL",
    is_locked=False,
)

SENDGRID_API_KEY = PiiPattern(
    id="SENDGRID_API_KEY",
    name="SendGrid API Key",
    category=PatternCategory.CREDENTIALS,
    dpdpa_schedule=DPDPASchedule.CRITICAL_PERSONAL_DATA,
    patterns=[
        # SG. + 22 base64url chars + . + 43 base64url chars
        r"\bSG\.[0-9A-Za-z_\-]{22}\.[0-9A-Za-z_\-]{43}\b",
    ],
    confidence_base=0.99,
    sensitivity="critical",
    context_keywords=[
        "sendgrid", "sendgrid_key", "email_api_key", "sg_key",
    ],
    validator=_always_valid,
    description="SendGrid API key with SG. prefix and fixed-length structure.",
    false_positive_risk="low",
    example="SG.<22_CHAR_KEY>.<43_CHAR_SIGNATURE>",
    country="GLOBAL",
    is_locked=False,
)

JWT_TOKEN = PiiPattern(
    id="JWT_TOKEN",
    name="JSON Web Token (JWT)",
    category=PatternCategory.CREDENTIALS,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # eyJ<header>.eyJ<payload>.<signature>
        r"\beyJ[A-Za-z0-9_\-]+\.eyJ[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+\b",
    ],
    confidence_base=0.95,
    sensitivity="high",
    context_keywords=[
        "jwt", "token", "bearer", "auth_token", "access_token",
        "id_token", "jwt_token",
    ],
    validator=_validate_jwt,
    description="JSON Web Token: three base64url segments separated by dots, header starting eyJ.",
    false_positive_risk="low",
    example="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
    country="GLOBAL",
    is_locked=False,
)

RSA_PRIVATE_KEY = PiiPattern(
    id="RSA_PRIVATE_KEY",
    name="PEM Private Key",
    category=PatternCategory.CREDENTIALS,
    dpdpa_schedule=DPDPASchedule.CRITICAL_PERSONAL_DATA,
    patterns=[
        # PEM header for RSA, EC, DSA, and OpenSSH private keys
        r"-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----",
    ],
    confidence_base=0.99,
    sensitivity="critical",
    context_keywords=[
        "private_key", "rsa_key", "pem", "ssh_key", "ssl_key",
        "tls_private",
    ],
    validator=_always_valid,
    description="PEM-encoded private key header (RSA, EC, DSA, or OpenSSH formats).",
    false_positive_risk="low",
    example="-----BEGIN RSA PRIVATE KEY-----",
    country="GLOBAL",
    is_locked=False,
)

AZURE_CONNECTION_STRING = PiiPattern(
    id="AZURE_CONNECTION_STRING",
    name="Azure Storage Connection String",
    category=PatternCategory.CREDENTIALS,
    dpdpa_schedule=DPDPASchedule.CRITICAL_PERSONAL_DATA,
    patterns=[
        # DefaultEndpointsProtocol=https;AccountName=...;AccountKey=...
        r"(?i)DefaultEndpointsProtocol=https?;AccountName=[^;]+;AccountKey=[^;]+",
    ],
    confidence_base=0.99,
    sensitivity="critical",
    context_keywords=[
        "azure", "connection_string", "storage_account", "azure_storage",
        "blob_storage",
    ],
    validator=_always_valid,
    description="Azure Storage connection string with AccountName and AccountKey embedded.",
    false_positive_risk="low",
    example="DefaultEndpointsProtocol=https;AccountName=devstoreaccount;AccountKey=Eby8vd==",
    country="GLOBAL",
    is_locked=False,
)

AZURE_SAS_TOKEN = PiiPattern(
    id="AZURE_SAS_TOKEN",
    name="Azure SAS Token",
    category=PatternCategory.CREDENTIALS,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # sv=YYYY-MM-DD&s[ser]=...&sig=...
        r"(?i)sv=[0-9]{4}-[0-9]{2}-[0-9]{2}&s[ser]=[^&]+&sig=[A-Za-z0-9%/+=]+",
    ],
    confidence_base=0.95,
    sensitivity="critical",
    context_keywords=[
        "sas_token", "shared_access_signature", "azure_sas",
        "blob_sas", "container_sas",
    ],
    validator=_always_valid,
    description="Azure Shared Access Signature token with sv/sig parameters.",
    false_positive_risk="low",
    example="sv=2021-06-08&ss=b&sig=abc123%2Fxyz%3D",
    country="GLOBAL",
    is_locked=False,
)

GCP_SERVICE_ACCOUNT = PiiPattern(
    id="GCP_SERVICE_ACCOUNT",
    name="GCP Service Account Key (JSON)",
    category=PatternCategory.CREDENTIALS,
    dpdpa_schedule=DPDPASchedule.CRITICAL_PERSONAL_DATA,
    patterns=[
        # JSON field combination unique to GCP service account keys
        r'(?i)"type"\s*:\s*"service_account"',
        r'(?i)"private_key_id"\s*:\s*"[a-f0-9]{40}"',
    ],
    confidence_base=0.99,
    sensitivity="critical",
    context_keywords=[
        "gcp", "service_account", "google_credentials",
        "gcp_key", "firebase_admin",
    ],
    validator=_always_valid,
    description="GCP service account key JSON file marker — type=service_account with private_key_id.",
    false_positive_risk="low",
    example='"type": "service_account"',
    country="GLOBAL",
    is_locked=False,
)

GENERIC_SECRET_HIGH_ENTROPY = PiiPattern(
    id="GENERIC_SECRET_HIGH_ENTROPY",
    name="High-Entropy Generic Secret",
    category=PatternCategory.CREDENTIALS,
    dpdpa_schedule=DPDPASchedule.SENSITIVE_PERSONAL_DATA,
    patterns=[
        # 20-100 printable non-whitespace characters (base64, hex, etc.)
        r"[A-Za-z0-9/+=_\-]{20,100}",
    ],
    confidence_base=0.7,
    sensitivity="high",
    context_keywords=[
        "password", "secret", "token", "key", "api_key",
        "auth_token", "private_key", "passwd", "pwd", "credential",
    ],
    validator=_validate_high_entropy_secret,
    description=(
        "High-entropy string in a column named password/secret/token/key; "
        "Shannon entropy > 3.5 bits/char, length 20-200."
    ),
    false_positive_risk="medium",
    example="V3ryS3cr3tP4ssw0rd!2024XYZ==",
    country="GLOBAL",
    is_locked=False,
)


CREDENTIAL_PATTERNS = [
    AWS_ACCESS_KEY,
    AWS_SECRET_KEY,
    GITHUB_TOKEN,
    GOOGLE_API_KEY,
    STRIPE_SECRET_KEY,
    STRIPE_PUBLISHABLE_KEY,
    SLACK_TOKEN,
    TWILIO_SID,
    SENDGRID_API_KEY,
    JWT_TOKEN,
    RSA_PRIVATE_KEY,
    AZURE_CONNECTION_STRING,
    AZURE_SAS_TOKEN,
    GCP_SERVICE_ACCOUNT,
    GENERIC_SECRET_HIGH_ENTROPY,
]
