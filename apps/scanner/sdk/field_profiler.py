"""
Field profiler for ARC-HAWK-DD scanner (P4-1).

Computes per-column statistics before classification:
  - null_rate:          null_count / total_count
  - uniqueness_ratio:   distinct_count / total_count
  - pattern_frequency:  pii_match_count / total_count  (filled in by caller after classification)
  - entropy_score:      average Shannon entropy of non-null values
  - value_distribution: top-10 most frequent values (anonymized — truncated to first 3 chars)
  - numeric stats:      min, max, mean, stddev (for numeric columns)

The profiling dict is attached to each finding as `profiling`.
"""

import math
import statistics
from collections import Counter
from typing import Any, Dict, List, Optional

from hawk_scanner.internals.entropy import calculate_shannon_entropy


def _is_numeric(value: Any) -> bool:
    try:
        float(str(value).replace(',', ''))
        return True
    except (ValueError, TypeError):
        return False


def _anonymize(value: str, keep_chars: int = 3) -> str:
    """Truncate to first `keep_chars` chars and mask the rest with *."""
    value = str(value)
    if len(value) <= keep_chars:
        return '*' * len(value)
    return value[:keep_chars] + '*' * (len(value) - keep_chars)


def profile_column(column_name: str, values: List[Any]) -> Dict[str, Any]:
    """
    Compute field-level statistics for a list of raw column values.

    Args:
        column_name: Column name (used for metadata only).
        values:      All values in this column (including nulls).

    Returns:
        Dict with profiling metrics. All values are JSON-serializable.
    """
    total = len(values)
    if total == 0:
        return {
            'column': column_name,
            'total_count': 0,
            'null_rate': 0.0,
            'uniqueness_ratio': 0.0,
            'entropy_score': 0.0,
            'value_distribution': [],
        }

    non_null = [v for v in values if v is not None and str(v).strip() != '']
    null_count = total - len(non_null)

    str_values = [str(v) for v in non_null]
    distinct = len(set(str_values))

    # Entropy: average over non-null sample (max 1000 values to cap cost)
    entropy_sample = str_values[:1000]
    entropy_score = (
        sum(calculate_shannon_entropy(v) for v in entropy_sample) / len(entropy_sample)
        if entropy_sample else 0.0
    )

    # Top-10 value distribution (anonymized)
    counter = Counter(str_values)
    top_10 = [
        {'value': _anonymize(v), 'count': c}
        for v, c in counter.most_common(10)
    ]

    null_rate = round(null_count / total, 4)
    uniqueness_ratio = round(distinct / total, 4) if total > 0 else 0.0

    # Phase 1: NULL-heavy columns — skip classification (>95% null = no signal)
    skip_classification = null_rate > 0.95

    # Phase 4: Low-entropy columns — all-identical values are noise, not PII
    low_entropy = entropy_score < 0.1 and len(non_null) > 10

    # Phase 1: Encryption detection — high entropy (>4.5 bits/char avg) suggests
    # already-encrypted or base64 content. Tag it rather than classifying PII.
    possibly_encrypted = entropy_score > 4.5 and uniqueness_ratio > 0.9

    # Phase 4: JSON-in-TEXT detection — check if values parse as JSON objects/arrays
    json_count = 0
    for v in str_values[:200]:
        stripped = v.strip()
        if stripped and stripped[0] in ('{', '['):
            try:
                import json as _json
                _json.loads(stripped)
                json_count += 1
            except Exception:
                pass
    json_in_text = json_count > len(str_values[:200]) * 0.5 if str_values else False

    # Phase 4: SHA-256 column fingerprint for dedup (skip unchanged files)
    col_fingerprint = None
    if str_values:
        import hashlib
        digest_input = "\n".join(sorted(str_values[:1000]))
        col_fingerprint = hashlib.sha256(digest_input.encode("utf-8", errors="replace")).hexdigest()[:16]

    result: Dict[str, Any] = {
        'column': column_name,
        'total_count': total,
        'null_rate': null_rate,
        'uniqueness_ratio': uniqueness_ratio,
        'entropy_score': round(entropy_score, 4),
        'value_distribution': top_10,
        # Hints for the classification layer
        'skip_classification': skip_classification,
        'low_entropy': low_entropy,
        'possibly_encrypted': possibly_encrypted,
        'json_in_text': json_in_text,
        'col_fingerprint': col_fingerprint,
    }
    if skip_classification:
        result['skip_reason'] = 'NULL_HEAVY'
    elif possibly_encrypted:
        result['skip_reason'] = 'POSSIBLE_ENCRYPTED'

    # Numeric stats
    numeric_vals: List[float] = []
    for v in non_null:
        if _is_numeric(v):
            try:
                numeric_vals.append(float(str(v).replace(',', '')))
            except ValueError:
                pass

    if numeric_vals:
        result['numeric'] = {
            'min': min(numeric_vals),
            'max': max(numeric_vals),
            'mean': round(statistics.mean(numeric_vals), 4),
            'stddev': round(statistics.stdev(numeric_vals), 4) if len(numeric_vals) > 1 else 0.0,
        }

    return result


def profile_table(columns: List[str], rows: List[tuple]) -> Dict[str, Dict[str, Any]]:
    """
    Profile all columns in a table given its rows.

    Args:
        columns: List of column names.
        rows:    List of row tuples (same order as columns).

    Returns:
        Dict mapping column_name → profiling dict.
    """
    # Transpose rows to column-oriented lists
    col_values: Dict[str, List[Any]] = {col: [] for col in columns}
    for row in rows:
        for col, val in zip(columns, row):
            col_values[col].append(val)

    return {col: profile_column(col, vals) for col, vals in col_values.items()}


def attach_profiling(finding: Dict[str, Any], profiling: Dict[str, Any], pii_count_by_col: Optional[Dict[str, int]] = None) -> Dict[str, Any]:
    """
    Attach profiling data to an existing finding dict, adding pattern_frequency.

    Args:
        finding:          Finding dict from scanner.
        profiling:        Output of profile_column for this finding's column.
        pii_count_by_col: How many PII matches found per column (for pattern_frequency).

    Returns:
        Finding with 'profiling' key added in-place.
    """
    col = finding.get('column', finding.get('file_path', ''))
    total = profiling.get('total_count', 0)

    enriched = dict(profiling)
    if pii_count_by_col and col and total > 0:
        pii_matches = pii_count_by_col.get(col, 0)
        enriched['pattern_frequency'] = round(pii_matches / total, 4)
    else:
        enriched['pattern_frequency'] = None

    finding['profiling'] = enriched
    return finding
