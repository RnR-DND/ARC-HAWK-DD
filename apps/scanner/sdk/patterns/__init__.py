"""
Hawk PII Pattern Library
========================
A world-class pattern library for the Hawk scanner covering India PII,
global identity documents, financial identifiers, credentials, and healthcare codes.

Quick start
-----------
    from sdk.patterns import get_pattern, get_by_category, ALL_PATTERNS
    from sdk.patterns.base import PiiPattern, PatternCategory, DPDPASchedule

    # Get a specific pattern
    aadhaar = get_pattern("IN_AADHAAR")

    # Get all India identity patterns
    india_id = get_by_category(PatternCategory.INDIA_IDENTITY)
"""

from .base import PiiPattern, PatternCategory, DPDPASchedule, ValidationResult
from .registry import (
    ALL_PATTERNS,
    get_pattern,
    get_by_category,
    get_locked_patterns,
    get_by_country,
    get_by_sensitivity,
    search_by_keyword,
    get_stats,
)

__all__ = [
    # Base types
    "PiiPattern",
    "PatternCategory",
    "DPDPASchedule",
    "ValidationResult",
    # Registry API
    "ALL_PATTERNS",
    "get_pattern",
    "get_by_category",
    "get_locked_patterns",
    "get_by_country",
    "get_by_sensitivity",
    "search_by_keyword",
    "get_stats",
]
