"""
Regression Test Suite for SDK Validators
==========================================
Tests all validators against ground truth data to ensure accuracy.

Target: 95%+ precision on Phase 1 PII types
"""

import sys
from pathlib import Path

import pytest
import yaml

# Add scanner to path
sys.path.insert(0, str(Path(__file__).parent.parent))

from sdk.validators import Verhoeff, Luhn, is_dummy_data


class TestVerhoeffAlgorithm:
    """Test Verhoeff implementation against known values."""

    VALID_CASES = [
        "234567890124",
        "999911112221",
    ]

    INVALID_CASES = [
        ("111111111111", "Repeating"),
        ("123456789012", "Sequential"),
        ("999911112222", "Bad checksum"),
    ]

    def test_accepts_valid_numbers(self):
        for number in self.VALID_CASES:
            assert Verhoeff.validate(number), f"Should accept {number}"

    def test_rejects_invalid_numbers(self):
        for number, reason in self.INVALID_CASES:
            assert not Verhoeff.validate(number), f"Should reject {number} ({reason})"


class TestLuhnAlgorithm:
    """Test Luhn implementation."""

    VALID_CARDS = [
        "4532015112830366",  # Visa
        "6011514433546201",  # Discover
    ]

    INVALID_CARDS = [
        "4532015112830367",  # Bad checksum
        "1111111111111111",  # Repeating
    ]

    def test_accepts_valid_cards(self):
        for card in self.VALID_CARDS:
            assert Luhn.validate(card), f"Should accept {card}"

    def test_rejects_invalid_cards(self):
        for card in self.INVALID_CARDS:
            assert not Luhn.validate(card), f"Should reject {card}"


class TestDummyDetector:
    """Test dummy data detection."""

    DUMMY_CASES = [
        "111111111111",
        "123456789012",
        "987654321098",
        "121212121212",
    ]

    REAL_CASES = [
        "999911112221",
        "234567890124",
    ]

    def test_detects_dummy_data(self):
        for data in self.DUMMY_CASES:
            assert is_dummy_data(data), f"Should detect dummy: {data}"

    def test_accepts_real_data(self):
        for data in self.REAL_CASES:
            assert not is_dummy_data(data), f"False positive on real data: {data}"


class TestGroundTruth:
    """Test against ground truth YAML."""

    @pytest.fixture
    def ground_truth_data(self):
        path = Path(__file__).parent / "ground_truth" / "phase1_test_data.yml"
        if not path.exists():
            pytest.skip(f"Ground truth file not found: {path}")
        with open(path, 'r') as f:
            return yaml.safe_load(f)

    def test_valid_aadhaar(self, ground_truth_data):
        for case in ground_truth_data.get('valid_aadhaar', []):
            number = case['number']
            clean = ''.join(c for c in number if c.isdigit())
            assert not is_dummy_data(clean) and Verhoeff.validate(clean), \
                f"Should detect: {case['description']}: {number}"

    def test_invalid_aadhaar(self, ground_truth_data):
        for case in ground_truth_data.get('invalid_aadhaar', []):
            number = case['number']
            clean = ''.join(c for c in number if c.isdigit())
            is_dummy = is_dummy_data(clean)
            is_valid = Verhoeff.validate(clean) if len(clean) == 12 else False
            assert is_dummy or not is_valid, \
                f"Should reject: {case['description']}: {number}"


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
