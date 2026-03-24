"""
SDK Snapshot Tests - Lock Baseline Behavior
Tests mathematical validators with known inputs/outputs
"""
import sys
from pathlib import Path
import pytest

# Add scanner root to path
sys.path.insert(0, str(Path(__file__).parent.parent))

from sdk.validators import validate_aadhaar, validate_credit_card, is_dummy_data
from sdk.validators import Verhoeff, Luhn


class TestVerhoeffSnapshot:
    """Snapshot tests for Verhoeff algorithm"""

    # Known valid Aadhaar numbers (with correct Verhoeff checksums)
    VALID_AADHAAR = [
        "234567890124",  # Valid checksum
        "999911112221",  # Valid checksum
    ]

    # Known invalid Aadhaar numbers
    INVALID_AADHAAR = [
        "123456789013",  # Wrong checksum
        "000000000000",  # All zeros (first digit 0)
        "111111111111",  # All ones (first digit 1)
        "999911112222",  # Off by one
    ]

    def test_valid_aadhaar_numbers(self):
        """Verhoeff must accept these valid numbers"""
        for number in self.VALID_AADHAAR:
            assert validate_aadhaar(number), f"Failed for {number}"

    def test_invalid_aadhaar_numbers(self):
        """Verhoeff must reject these invalid numbers"""
        for number in self.INVALID_AADHAAR:
            assert not validate_aadhaar(number), f"Should reject {number}"

    def test_length_validation(self):
        """Verhoeff must reject wrong-length inputs"""
        assert not validate_aadhaar("12345")  # Too short
        assert not validate_aadhaar("1234567890123")  # Too long

    def test_formatted_inputs(self):
        """validate_aadhaar strips spaces/hyphens before validation"""
        # Generate a known valid number for formatted test
        for number in self.VALID_AADHAAR:
            spaced = f"{number[:4]} {number[4:8]} {number[8:]}"
            assert validate_aadhaar(spaced), f"Spaced format failed: {spaced}"


class TestLuhnSnapshot:
    """Snapshot tests for Luhn algorithm"""

    # Known valid credit card numbers (test cards)
    VALID_CARDS = [
        "4532015112830366",  # Visa
        "5425233430109903",  # Mastercard
        "374245455400126",   # Amex (15 digits)
        "6011000991300009",  # Discover
    ]

    # Known invalid card numbers
    INVALID_CARDS = [
        "4532015112830367",  # Wrong checksum
        "0000000000000000",  # All zeros
        "1234567890123456",  # Sequential
    ]

    def test_valid_card_numbers(self):
        """Luhn must accept these valid cards"""
        for card in self.VALID_CARDS:
            assert validate_credit_card(card), f"Failed for {card}"

    def test_invalid_card_numbers(self):
        """Luhn must reject these invalid cards"""
        for card in self.INVALID_CARDS:
            assert not validate_credit_card(card), f"Should reject {card}"

    def test_length_variations(self):
        """Luhn works with different card lengths"""
        assert validate_credit_card("374245455400126")  # 15 digits (Amex)
        assert validate_credit_card("4532015112830366")  # 16 digits (Visa)


class TestDummyDetectorSnapshot:
    """Snapshot tests for dummy data detection"""

    # Patterns that MUST be detected as dummy
    DUMMY_PATTERNS = [
        "123456789012",  # Sequential
        "111111111111",  # Repeating
        "000000000000",  # All zeros
        "999999999999",  # All nines
        "121212121212",  # Alternating
        "123412341234",  # Repeated block
    ]

    # Patterns that are NOT dummy (look random)
    REAL_PATTERNS = [
        "234567890124",  # Valid Aadhaar-like
        "4532015112830366",  # Valid card
    ]

    def test_detects_dummy_patterns(self):
        """Must catch all dummy patterns"""
        for pattern in self.DUMMY_PATTERNS:
            clean = pattern.replace(" ", "")
            assert is_dummy_data(clean), f"Missed dummy: {pattern}"

    def test_allows_real_patterns(self):
        """Must NOT flag real data as dummy"""
        for pattern in self.REAL_PATTERNS:
            clean = pattern.replace(" ", "")
            assert not is_dummy_data(clean), f"False positive: {pattern}"


class TestCombinedValidation:
    """Test the full validation pipeline"""

    def test_valid_aadhaar_full_check(self):
        """Valid Aadhaar passes all checks"""
        number = "234567890124"
        assert not is_dummy_data(number)
        assert validate_aadhaar(number)

    def test_dummy_aadhaar_rejected(self):
        """Dummy Aadhaar fails even if Verhoeff passes"""
        number = "123456789012"
        assert is_dummy_data(number)

    def test_valid_card_full_check(self):
        """Valid card passes all checks"""
        card = "4532015112830366"
        assert not is_dummy_data(card)
        assert validate_credit_card(card)


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
