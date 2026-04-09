"""
PII Pattern Library — Base Types
=================================
Core dataclass, enums, and shared types for the Hawk pattern library.

All patterns must be RE2-safe (no catastrophic backtracking).
"""

from dataclasses import dataclass, field
from enum import Enum
from typing import Callable, Optional
import re


class PatternCategory(str, Enum):
    INDIA_IDENTITY = "india_identity"
    INDIA_FINANCIAL = "india_financial"
    INDIA_CORPORATE = "india_corporate"
    INDIA_HEALTHCARE = "india_healthcare"
    GLOBAL_PII = "global_pii"
    FINANCIAL_GLOBAL = "financial_global"
    CREDENTIALS = "credentials"
    PERSONAL = "personal"
    HEALTHCARE_GLOBAL = "healthcare_global"


class DPDPASchedule(str, Enum):
    PERSONAL_DATA = "Personal Data"
    SENSITIVE_PERSONAL_DATA = "Sensitive Personal Data"
    CRITICAL_PERSONAL_DATA = "Critical Personal Data"


class ValidationResult(str, Enum):
    VALID = "valid"
    INVALID = "invalid"
    UNCERTAIN = "uncertain"   # Regex matched but validator not available


@dataclass
class PiiPattern:
    """
    Descriptor for a single PII pattern in the Hawk library.

    Fields
    ------
    id                  Unique uppercase identifier, e.g. "IN_AADHAAR", "US_SSN".
    name                Human-readable display name.
    category            PatternCategory enum value.
    dpdpa_schedule      DPDPA 2023 schedule classification.
    patterns            List of RE2-safe regex strings (at least one).
    confidence_base     Base confidence score 0.0–1.0 before context adjustment.
    sensitivity         "critical" | "high" | "medium" | "low"
    context_keywords    Column/table name substrings that boost confidence.
    validator           Optional callable (str) -> bool for math/format check.
    description         One-sentence description for documentation.
    false_positive_risk "low" | "medium" | "high" | "critical"
    example             A syntactically valid example value.
    country             ISO-3166-1 alpha-2 or "GLOBAL".
    is_locked           If True, cannot be disabled by user configuration.
    """
    id: str
    name: str
    category: PatternCategory
    dpdpa_schedule: DPDPASchedule
    patterns: list  # list[str]  — RE2-safe regexes
    confidence_base: float
    sensitivity: str
    context_keywords: list  # list[str]
    validator: Optional[Callable] = None
    description: str = ""
    false_positive_risk: str = "medium"
    example: str = ""
    country: str = "IN"
    is_locked: bool = False

    # ------------------------------------------------------------------ helpers

    def compile(self):
        """Return list of compiled re.Pattern objects."""
        return [re.compile(p) for p in self.patterns]

    def boost_confidence(self, column_name: str, table_name: str = "") -> float:
        """
        Return adjusted confidence when column/table context is considered.

        Args:
            column_name: Normalised (lowercased) column name.
            table_name:  Normalised (lowercased) table name (optional).

        Returns:
            Adjusted confidence clamped to [0.0, 1.0].
        """
        combined = f"{column_name} {table_name}".lower()
        for kw in self.context_keywords:
            if kw.lower() in combined:
                # Each matching keyword adds 0.15, capped at 1.0
                boosted = min(1.0, self.confidence_base + 0.15)
                return boosted
        return self.confidence_base

    def validate(self, value: str) -> ValidationResult:
        """
        Run the pattern's validator if available.

        Returns:
            ValidationResult enum value.
        """
        if self.validator is None:
            return ValidationResult.UNCERTAIN
        try:
            result = self.validator(value)
            return ValidationResult.VALID if result else ValidationResult.INVALID
        except Exception:
            return ValidationResult.UNCERTAIN

    def __repr__(self) -> str:
        return (
            f"PiiPattern(id={self.id!r}, category={self.category.value!r}, "
            f"confidence={self.confidence_base}, locked={self.is_locked})"
        )
