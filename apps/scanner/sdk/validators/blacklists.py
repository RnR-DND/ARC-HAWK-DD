"""
Email Domain Blacklist - Filter Test/Invalid Email Domains
==========================================================
Blacklist of domains commonly used for test data or invalid emails.

Usage:
    from sdk.validators.blacklists import is_blacklisted_domain
    
    if is_blacklisted_domain("test@test.com"):
        # Reject as test data
"""

from typing import Set

# Test/dummy domains that should be rejected
TEST_DOMAINS: Set[str] = {
    # Generic test domains
    'test.com',
    'example.com',
    'example.org',
    'example.net',
    'dummy.com',
    'sample.com',
    'fake.com',
    'mock.com',
    
    # Development/local domains
    'localhost',
    'test.local',
    'dev.local',
    'local',
    '127.0.0.1',
    
    # Disposable email domains (common ones)
    'mailinator.com',
    'guerrillamail.com',
    'temp-mail.org',
    '10minutemail.com',
    'throwaway.email',
    
    # Invalid TLDs
    'test',
    'invalid',
    'localhost',
}

# Additional patterns to check
SUSPICIOUS_PATTERNS = [
    'test',      # Contains 'test'
    'dummy',     # Contains 'dummy'
    'fake',      # Contains 'fake'
    'sample',    # Contains 'sample'
    'example',   # Contains 'example'
]


def is_blacklisted_domain(email: str) -> bool:
    """
    Check if an email domain is blacklisted.
    
    Args:
        email: Email address to check
        
    Returns:
        True if domain is blacklisted, False otherwise
    """
    if not email or '@' not in email:
        return False
    
    # Extract domain
    domain = email.split('@')[-1].lower().strip()
    
    # Check exact match
    if domain in TEST_DOMAINS:
        return True
    
    # Check suspicious patterns
    for pattern in SUSPICIOUS_PATTERNS:
        if pattern in domain:
            return True
    
    return False


def is_valid_email_domain(email: str) -> bool:
    """
    Check if an email has a valid (non-blacklisted) domain.
    
    Args:
        email: Email address to check
        
    Returns:
        True if domain is valid, False if blacklisted
    """
    return not is_blacklisted_domain(email)


if __name__ == "__main__":
    print("=== Email Domain Blacklist Tests ===\n")
    
    test_cases = [
        ("test@test.com", True, "test.com domain"),
        ("user@example.com", True, "example.com domain"),
        ("dummy@dummy.com", True, "dummy.com domain"),
        ("john@company.com", False, "Valid company domain"),
        ("user@gmail.com", False, "Gmail domain"),
        ("admin@localhost", True, "localhost domain"),
        ("test@mailinator.com", True, "Disposable email"),
        ("user@testdomain.com", True, "Contains 'test'"),
        ("real@production.com", False, "Valid production domain"),
    ]
    
    for email, expected_blacklisted, description in test_cases:
        is_blacklisted = is_blacklisted_domain(email)
        status = "✓" if is_blacklisted == expected_blacklisted else "✗"
        result_str = "BLACKLISTED" if is_blacklisted else "VALID"
        print(f"{status} {description}: {email}")
        print(f"   Expected: {'BLACKLISTED' if expected_blacklisted else 'VALID'}, Got: {result_str}\n")
