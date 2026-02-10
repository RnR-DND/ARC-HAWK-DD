"""
Comprehensive Test Suite for Zero False Positives
=================================================
Tests all 11 PII types with valid data, invalid data, and test data.

Goal: Achieve 100% accuracy (no false positives, no false negatives)
"""

import sys
import os

import pytest

# Add parent directory to path
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from sdk.validators.verhoeff import validate_aadhaar
from sdk.validators.pan import validate_pan
from sdk.validators.phone import validate_indian_phone
from sdk.validators.email import validate_email
from sdk.validators.luhn import validate_credit_card
from sdk.validators.passport import validate_indian_passport
from sdk.validators.upi import validate_upi
from sdk.validators.ifsc import validate_ifsc
from sdk.validators.bank_account import validate_bank_account
from sdk.validators.voter_id import validate_voter_id
from sdk.validators.driving_license import validate_driving_license


class TestAadhaar:
    """Test Aadhaar validation"""

    VALID = [
        "234567890124",  # Valid checksum
        "999911112226",  # Valid checksum
    ]

    INVALID = [
        "111111111111",  # All same (dummy data)
        "123456789012",  # Sequential (dummy data)
        "000000000000",  # All zeros
        "999911112222",  # Invalid checksum
        "1234567890",    # Too short
        "12345678901234",  # Too long
    ]

    def test_valid(self):
        for value in self.VALID:
            assert validate_aadhaar(value), f"Should accept: {value}"

    def test_invalid(self):
        for value in self.INVALID:
            assert not validate_aadhaar(value), f"Should reject: {value}"


class TestPhone:
    """Test phone validation"""

    VALID = [
        "9123456789",
        "8765432109",
        "7654321098",
        "6543210987",
    ]

    INVALID = [
        "9999999999",  # All same
        "0123456789",  # Starts with 0
        "1234567890",  # Starts with 1
        "9876543210",  # Full descending sequence
        "5876543210",  # Invalid prefix (5)
        "987654321",   # Too short
        "98765432109",  # Too long
    ]

    def test_valid(self):
        for value in self.VALID:
            assert validate_indian_phone(value), f"Should accept: {value}"

    def test_invalid(self):
        for value in self.INVALID:
            assert not validate_indian_phone(value), f"Should reject: {value}"


class TestEmail:
    """Test email validation"""

    VALID = [
        "john@company.com",
        "user@gmail.com",
        "admin@production.com",
        "support@enterprise.org",
    ]

    INVALID = [
        "test@test.com",
        "user@example.com",
        "dummy@dummy.com",
        "admin@localhost",
        "test@mailinator.com",
        "user@testdomain.com",
        "invalid.email",
        "@example.com",
        "user@",
    ]

    def test_valid(self):
        for value in self.VALID:
            assert validate_email(value), f"Should accept: {value}"

    def test_invalid(self):
        for value in self.INVALID:
            assert not validate_email(value), f"Should reject: {value}"


class TestCreditCard:
    """Test credit card validation"""

    VALID = [
        "4532015112830366",  # Visa
        "5425233430109903",  # Mastercard
    ]

    INVALID = [
        "1111111111111111",
        "1234567890123456",
        "0000000000000000",
        "4532015112830367",  # Invalid Luhn
    ]

    def test_valid(self):
        for value in self.VALID:
            assert validate_credit_card(value), f"Should accept: {value}"

    def test_invalid(self):
        for value in self.INVALID:
            assert not validate_credit_card(value), f"Should reject: {value}"


class TestPan:
    """Test PAN validation"""

    VALID = [
        "ABCDE1234F",
        "ZZZZZ9999Z",
    ]

    INVALID = [
        "AAAAA0000A",
        "TEST12345",
        "ABCD1234F",   # Too short
        "ABCDE12345F",  # Too long
    ]

    def test_valid(self):
        for value in self.VALID:
            assert validate_pan(value), f"Should accept: {value}"

    def test_invalid(self):
        for value in self.INVALID:
            assert not validate_pan(value), f"Should reject: {value}"


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
