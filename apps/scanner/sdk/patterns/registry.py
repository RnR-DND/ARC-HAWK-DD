"""
PII Pattern Registry
=====================
Master registry of all patterns in the Hawk pattern library.

Usage
-----
    from sdk.patterns.registry import get_pattern, get_by_category, ALL_PATTERNS

    # Look up a single pattern
    p = get_pattern("IN_AADHAAR")

    # All India identity patterns
    india = get_by_category(PatternCategory.INDIA_IDENTITY)

    # Patterns that are locked (cannot be disabled)
    locked = get_locked_patterns()

    # Free-text search
    results = search_by_keyword("aadhaar")
"""

from typing import Optional

from .base import PiiPattern, PatternCategory

# ── Import all sub-libraries ──────────────────────────────────────────────── #
# FIX H8: Changed bare 'from patterns.X import' to PEP-328 relative imports
from .india_identity import INDIA_IDENTITY_PATTERNS
from .india_financial import INDIA_FINANCIAL_PATTERNS
from .india_corporate import INDIA_CORPORATE_PATTERNS
from .india_healthcare import INDIA_HEALTHCARE_PATTERNS
from .global_pii import GLOBAL_PII_PATTERNS
from .financial_global import FINANCIAL_GLOBAL_PATTERNS
from .credentials import CREDENTIAL_PATTERNS
from .personal import PERSONAL_PATTERNS
from .healthcare_global import HEALTHCARE_GLOBAL_PATTERNS


# ── Build the master dictionary ───────────────────────────────────────────── #

def _build_registry() -> dict:
    """Combine all pattern lists into {id: PiiPattern} dict."""
    all_lists = [
        INDIA_IDENTITY_PATTERNS,
        INDIA_FINANCIAL_PATTERNS,
        INDIA_CORPORATE_PATTERNS,
        INDIA_HEALTHCARE_PATTERNS,
        GLOBAL_PII_PATTERNS,
        FINANCIAL_GLOBAL_PATTERNS,
        CREDENTIAL_PATTERNS,
        PERSONAL_PATTERNS,
        HEALTHCARE_GLOBAL_PATTERNS,
    ]
    registry = {}
    for lst in all_lists:
        for pattern in lst:
            if pattern.id in registry:
                raise ValueError(
                    f"Duplicate pattern ID in registry: {pattern.id!r}. "
                    "Each pattern must have a unique ID."
                )
            registry[pattern.id] = pattern
    return registry


# Singleton registry — built once at import time
ALL_PATTERNS: dict = _build_registry()


# ── Public API ────────────────────────────────────────────────────────────── #

def get_pattern(pii_id: str) -> Optional[PiiPattern]:
    """
    Look up a single pattern by its ID (case-insensitive).

    Args:
        pii_id: Pattern identifier, e.g. "IN_AADHAAR", "US_SSN".

    Returns:
        PiiPattern or None if not found.
    """
    return ALL_PATTERNS.get(pii_id.upper().strip())


def get_by_category(category: PatternCategory) -> list:
    """
    Return all patterns belonging to a given category.

    Args:
        category: PatternCategory enum value.

    Returns:
        List of PiiPattern objects (may be empty).
    """
    return [p for p in ALL_PATTERNS.values() if p.category == category]


def get_locked_patterns() -> list:
    """
    Return all patterns marked as locked (cannot be disabled by users).

    Returns:
        List of locked PiiPattern objects.
    """
    return [p for p in ALL_PATTERNS.values() if p.is_locked]


def get_by_country(country_code: str) -> list:
    """
    Return all patterns for a given ISO-3166-1 country code or "GLOBAL".

    Args:
        country_code: e.g. "IN", "US", "GB", "GLOBAL".

    Returns:
        List of PiiPattern objects.
    """
    code = country_code.upper().strip()
    return [p for p in ALL_PATTERNS.values() if p.country == code]


def get_by_sensitivity(sensitivity: str) -> list:
    """
    Return all patterns at a given sensitivity level.

    Args:
        sensitivity: "critical" | "high" | "medium" | "low".

    Returns:
        List of PiiPattern objects.
    """
    level = sensitivity.lower().strip()
    return [p for p in ALL_PATTERNS.values() if p.sensitivity == level]


def search_by_keyword(keyword: str) -> list:
    """
    Free-text search across pattern id, name, description, and context_keywords.

    Args:
        keyword: Search term (case-insensitive).

    Returns:
        List of matching PiiPattern objects.
    """
    kw = keyword.lower().strip()
    matches = []
    for p in ALL_PATTERNS.values():
        haystack = " ".join([
            p.id.lower(),
            p.name.lower(),
            p.description.lower(),
            " ".join(p.context_keywords).lower(),
        ])
        if kw in haystack:
            matches.append(p)
    return matches


def get_stats() -> dict:
    """
    Return summary statistics about the registry.

    Returns:
        Dict with counts by category, sensitivity, country, and lock status.
    """
    stats = {
        "total": len(ALL_PATTERNS),
        "locked": len(get_locked_patterns()),
        "by_category": {},
        "by_sensitivity": {},
        "by_country": {},
    }
    for p in ALL_PATTERNS.values():
        cat = p.category.value
        stats["by_category"][cat] = stats["by_category"].get(cat, 0) + 1

        sens = p.sensitivity
        stats["by_sensitivity"][sens] = stats["by_sensitivity"].get(sens, 0) + 1

        country = p.country
        stats["by_country"][country] = stats["by_country"].get(country, 0) + 1

    return stats


if __name__ == "__main__":
    print("=== Hawk PII Pattern Registry ===\n")

    stats = get_stats()
    print(f"Total patterns:  {stats['total']}")
    print(f"Locked patterns: {stats['locked']}")

    print("\nBy category:")
    for cat, count in sorted(stats["by_category"].items()):
        print(f"  {cat:<25} {count}")

    print("\nBy sensitivity:")
    for sens, count in sorted(stats["by_sensitivity"].items()):
        print(f"  {sens:<10} {count}")

    print("\nBy country:")
    for country, count in sorted(stats["by_country"].items()):
        print(f"  {country:<10} {count}")

    print("\nLocked patterns (cannot be disabled):")
    for p in sorted(get_locked_patterns(), key=lambda x: x.id):
        print(f"  {p.id}")

    print("\nSearch test — keyword 'aadhaar':")
    for p in search_by_keyword("aadhaar"):
        print(f"  {p.id}: {p.name}")
