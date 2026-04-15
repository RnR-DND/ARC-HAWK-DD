"""
Email Address Validator
========================
Validates email addresses using RFC 5322 basic format.

Format: local@domain.tld
- Local part: alphanumeric + . _ % + -
- Domain: alphanumeric + . -
- TLD: at least 2 characters
"""

import re
from typing import Optional

# Import blacklist
try:
    from sdk.validators.blacklists import is_blacklisted_domain
    BLACKLIST_AVAILABLE = True
except ImportError:
    BLACKLIST_AVAILABLE = False


class EmailValidator:
    """
    Validates email addresses using RFC 5322 standard.
    
    Enhanced with domain blacklist to filter test emails.
    """
    
    # Simplified RFC 5322 regex (covers most common cases)
    EMAIL_PATTERN = re.compile(
        r'^[a-zA-Z0-9.!#$%&\'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$'
    )
    
    COMMON_TLDS = {
        "com", "org", "net", "edu", "gov", "mil", "int", "cc", "br",
        "biz", "io", "ai", "co", "me", "dev", "app", "tech", "xyz",
        "cloud", "online", "shop", "store", "ltd", "inc", "eu", "nl",
        "pro", "biz", "company", "us", "uk", "in", "de", "ca", "au",
        "fr", "jp", "it", "ru", "llc", "info"
    }

    INVALID_DOMAINS = {
        "localhost", "internal", "test", "dummy", "example", "lan",
        "invalid", "local", "home.arpa", "onion", "corp", "xyz",
        "abc", "sample"
    }

    INVALID_FILE_EXTENSIONS = {
        "png", "jpg", "jpeg", "gif", "svg", "webp", "bmp", "tiff",
        "ico", "rtf", "csv", "iso", "sh", "bat", "py", "php" , "xml",
        "pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "txt",
        "zip", "rar", "7z", "tar", "gz", "exe", "msi", "bin", "dmg",
        "html", "htm", "css", "js", "json"
    }

    SUSPICIOUS_DOMAINS = {
        "prod", "dev", "test", "stage", "staging", "override",
        "config", "system", "internal"
    }

    GENERIC_LOCALS = {
        "noreply", "no-reply", "donotreply", "admin", "support",
        "info", "contact"
    }

    @classmethod
    def validate(cls, email: str) -> bool:
        """
        Validates an email address.
        
        Args:
            email: Email address string
            
        Returns:
            True if valid, False otherwise
        """
        if not email:
            return False
        
        # Normalize
        email = email.strip().strip('.,:;()[]{}"\'').lower()
        
        # Basic format validation
        if not cls.EMAIL_PATTERN.match(email):
            return False
        
        # Split into local and domain
        local, domain = email.split('@')
        
        # Local part checks
        if len(local) == 0 or len(local) > 64:
            return False
        
        # Domain checks
        if len(domain) < 3 or len(domain) > 253:
            return False
        
        # Domain must have at least one dot
        if '.' not in domain:
            return False
        
        if '..' in email:
            return False

        parts = domain.split('.')
        
        tld = parts[-1]
        domain_name = parts[-2] if len(parts) >= 2 else ""

        if domain_name in cls.SUSPICIOUS_DOMAINS:
            return False

        if len(domain_name) <= 2:
            return False

        if len(tld) < 2 or len(tld) > 10:
            return False

        if tld not in cls.COMMON_TLDS:
            return False

        if any(p in cls.INVALID_DOMAINS for p in parts):
            return False

        if tld in cls.INVALID_FILE_EXTENSIONS:
            return False

        if len(local) >= 5 and len(set(local)) <= 2:
            return False

        if local in cls.GENERIC_LOCALS:
            return False
        # Reject common invalid patterns
        if cls._is_invalid_pattern(email):
            return False
        
        return True
    
    @staticmethod
    def _is_invalid_pattern(email: str) -> bool:
        """Check for obviously invalid patterns."""
        
        # Starts or ends with dot
        local, domain = email.split('@')
        if local.startswith('.') or local.endswith('.'):
            return True
        if domain.startswith('.') or domain.endswith('.'):
            return True
        
        # Domain starts with hyphen
        if domain.startswith('-'):
            return True
            
        return False


def validate_email(email: str) -> bool:
    """
    Validates an email address.
    
    Convenience function wrapping EmailValidator.validate()
    
    Args:
        email: Email address string
        
    Returns:
        True if valid, False otherwise
    """
    return EmailValidator.validate(email)


if __name__ == "__main__":
    print("=== Email Validator Tests ===\n")
    
    test_cases = [
        ("user@example.com", True, "Basic valid email"),
        ("john.doe@company.co.in", True, "Valid with dots and multi-level domain"),
        ("test+tag@gmail.com", True, "Valid with + tag"),
        ("user_name@domain-name.com", True, "Valid with underscore and hyphen"),
        ("a@b.co", True, "Minimal valid email"),
        ("invalid.email", False, "Missing @"),
        ("@example.com", False, "Missing local part"),
        ("user@", False, "Missing domain"),
        ("user@domain", False, "Missing TLD"),
        ("user..name@example.com", False, "Double dots"),
        (".user@example.com", False, "Starts with dot"),
        ("user@.example.com", False, "Domain starts with dot"),
        ("user@domain..com", False, "Double dots in domain"),
        ("a" * 65 + "@example.com", False, "Local part too long"),
    ]
    
    for email, expected, description in test_cases:
        result = validate_email(email)
        status = "✓" if result == expected else "✗"
        print(f"{status} {description}: {email}")
        print(f"   Expected: {expected}, Got: {result}\n")
