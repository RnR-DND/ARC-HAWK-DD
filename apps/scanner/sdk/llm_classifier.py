"""
Layer 3 LLM Classifier — Anthropic Claude API integration for contextual PII classification.

Called when:
  - Layer 2 (Presidio) returns a confidence score in the ambiguous band [0.65, 0.80], OR
  - The candidate is a DPDPA-specific sensitive category with no Presidio recognizer
    (Caste, Sexual Orientation, Political Opinion, Religious Belief).

Features:
  - Batches up to 20 findings per Claude prompt (20x round-trip reduction)
  - Redis caching keyed on sha256(column_name + pattern_name + value_fingerprint)
  - Per-scan budget cap (default: 500 LLM calls max per scan)
  - Graceful fallback: on any API error, returns the Layer 1/2 result with
    confidence capped at 0.64 and classifier tagged as "layer2_fallback"
  - Never logs or sends raw PII values — values are anonymized before the prompt
"""

import hashlib
import json
import logging
import os
import re
import time
from typing import Any

import anthropic

logger = logging.getLogger("arc-hawk.llm-classifier")

# DPDPA-specific sensitive categories that Presidio has no recognizer for.
# These are always routed to Claude (Layer 3) regardless of confidence band.
DPDPA_SENSITIVE_CATEGORIES = {
    "CASTE",
    "SEXUAL_ORIENTATION",
    "POLITICAL_OPINION",
    "RELIGIOUS_BELIEF",
    "POLITICAL_AFFILIATION",
    "TRIBE",
    "ETHNIC_ORIGIN",
}

# DPDPA 2023 Schedule categories used in the classification prompt.
DPDPA_CATEGORIES = [
    "Name",
    "DOB",
    "Address",
    "Phone",
    "Email",
    "Financial ID",
    "Health Record",
    "Biometric Reference",
    "Political Opinion",
    "Religious Belief",
    "Caste",
    "Sexual Orientation",
    "Contact Information",
    "Government ID",
    "Not PII",
]

_CONFIDENCE_AMBIGUOUS_LOW = float(os.getenv("LLM_CONFIDENCE_LOW", "0.65"))
_CONFIDENCE_AMBIGUOUS_HIGH = float(os.getenv("LLM_CONFIDENCE_HIGH", "0.80"))
_BATCH_SIZE = int(os.getenv("LLM_BATCH_SIZE", "20"))
_BUDGET_PER_SCAN = int(os.getenv("LLM_BUDGET_PER_SCAN", "500"))
_MODEL = os.getenv("LLM_MODEL", "claude-haiku-4-5-20251001")  # Haiku for cost efficiency
_CACHE_TTL = int(os.getenv("LLM_CACHE_TTL", str(7 * 86400)))  # 7 days


class LLMClassifier:
    """
    Layer 3 contextual PII classifier backed by Anthropic Claude.

    Usage:
        classifier = LLMClassifier(redis_client)
        results = classifier.classify_batch(findings, scan_id)
    """

    def __init__(self, redis_client=None):
        api_key = os.getenv("ANTHROPIC_API_KEY")
        if api_key:
            self._client = anthropic.Anthropic(api_key=api_key)
            self._available = True
            logger.info(f"LLM classifier initialized (model={_MODEL})")
        else:
            self._client = None
            self._available = False
            logger.warning("ANTHROPIC_API_KEY not set — Layer 3 LLM classifier disabled")
        self._redis = redis_client

    def should_invoke(self, finding: dict) -> bool:
        """Return True if this finding warrants a Layer 3 LLM call."""
        if not self._available:
            return False
        confidence = finding.get("confidence_score", 0.0) or 0.0
        category = (finding.get("dpdpa_category") or "").upper()
        # Always invoke for sensitive categories regardless of confidence
        if category in DPDPA_SENSITIVE_CATEGORIES:
            return True
        # Invoke for ambiguous band
        return _CONFIDENCE_AMBIGUOUS_LOW <= confidence <= _CONFIDENCE_AMBIGUOUS_HIGH

    def classify_batch(self, findings: list[dict], scan_id: str = "") -> list[dict]:
        """
        Classify a list of findings using Claude, respecting the per-scan budget.

        Returns the same list with updated `pii_category`, `dpdpa_schedule`,
        `confidence_score`, and `classifier` fields for each finding.
        """
        if not self._available or not findings:
            return findings

        results = list(findings)
        budget_remaining = _BUDGET_PER_SCAN
        call_count = 0

        # Split into batches of _BATCH_SIZE
        for batch_start in range(0, len(findings), _BATCH_SIZE):
            if budget_remaining <= 0:
                logger.warning(
                    f"[scan={scan_id}] LLM budget exhausted ({_BUDGET_PER_SCAN} calls). "
                    f"{len(findings) - batch_start} findings left at Layer 1/2."
                )
                break

            batch = findings[batch_start : batch_start + _BATCH_SIZE]
            cache_hits, uncached = self._check_cache(batch)

            # Apply cache hits immediately
            for idx, result in cache_hits.items():
                results[batch_start + idx] = {**results[batch_start + idx], **result}

            if not uncached:
                continue

            if budget_remaining < len(uncached):
                uncached = uncached[:budget_remaining]

            try:
                llm_results = self._call_claude(uncached, scan_id)
                call_count += 1
                budget_remaining -= len(uncached)

                for finding, llm_result in zip(uncached, llm_results):
                    idx = findings.index(finding)
                    merged = {**results[idx], **llm_result, "classifier": "llm"}
                    results[idx] = merged
                    self._write_cache(finding, llm_result)

            except Exception as exc:
                logger.error(f"[scan={scan_id}] Claude API error: {exc} — falling back to Layer 1/2")
                for finding in uncached:
                    idx = findings.index(finding)
                    # Cap confidence below "confirmed" threshold and tag as fallback
                    original = results[idx]
                    capped = min(original.get("confidence_score", 0.0) or 0.0, 0.64)
                    results[idx] = {
                        **original,
                        "confidence_score": capped,
                        "classifier": "layer2_fallback",
                    }

        if call_count:
            logger.info(f"[scan={scan_id}] Layer 3 LLM: {call_count} API calls, "
                        f"{budget_remaining} budget remaining")
        return results

    # ------------------------------------------------------------------
    # Private helpers
    # ------------------------------------------------------------------

    def _cache_key(self, finding: dict) -> str:
        """Deterministic cache key based on (column_name, pattern_name, value_fingerprint)."""
        col = (finding.get("column") or finding.get("file_path") or "").lower()
        pattern = (finding.get("pattern_name") or "").lower()
        # Fingerprint the first match value (never the raw value, just a hash)
        matches = finding.get("matches") or []
        val = matches[0] if matches else finding.get("sample_text", "")
        val_fp = hashlib.sha256(str(val).encode()).hexdigest()[:16]
        key_str = f"llm:v1:{col}:{pattern}:{val_fp}"
        return hashlib.sha256(key_str.encode()).hexdigest()

    def _check_cache(self, batch: list[dict]) -> tuple[dict, list[dict]]:
        """Return (cache_hits dict, uncached list)."""
        hits: dict[int, dict] = {}
        uncached: list[dict] = []
        for i, finding in enumerate(batch):
            if not self._redis:
                uncached.append(finding)
                continue
            key = self._cache_key(finding)
            try:
                cached = self._redis.get(f"llm_cache:{key}")
                if cached:
                    hits[i] = json.loads(cached)
                    continue
            except Exception:
                pass
            uncached.append(finding)
        return hits, uncached

    def _write_cache(self, finding: dict, result: dict) -> None:
        if not self._redis:
            return
        key = self._cache_key(finding)
        try:
            self._redis.setex(f"llm_cache:{key}", _CACHE_TTL, json.dumps(result))
        except Exception:
            pass

    def _anonymize(self, finding: dict, max_samples: int = 3) -> list[str]:
        """Return anonymized sample values — mask middle characters."""
        matches = finding.get("matches") or []
        samples = []
        for v in matches[:max_samples]:
            v = str(v)
            if len(v) <= 4:
                samples.append("***")
            else:
                masked = v[:2] + "*" * (len(v) - 4) + v[-2:]
                samples.append(masked)
        return samples

    def _call_claude(self, findings: list[dict], scan_id: str) -> list[dict]:
        """
        Send a single batched prompt to Claude and parse the structured response.
        Each finding gets one classification result.
        """
        items = []
        for i, f in enumerate(findings):
            col = f.get("column") or f.get("file_path", "")
            pattern = f.get("pattern_name", "")
            samples = self._anonymize(f)
            items.append(
                f"  {i+1}. column='{col}', pattern='{pattern}', "
                f"samples={samples}"
            )

        categories_str = ", ".join(f'"{c}"' for c in DPDPA_CATEGORIES)
        items_str = "\n".join(items)

        prompt = f"""You are a DPDPA 2023 (India) data privacy classifier. For each field below, classify it into EXACTLY ONE of these categories: {categories_str}.

Fields to classify:
{items_str}

Respond with a JSON array of objects, one per field, in order:
[
  {{"index": 1, "category": "...", "confidence": 0.0-1.0, "is_pii": true/false}},
  ...
]

Rules:
- Use the exact category strings provided
- confidence: your certainty 0.0-1.0
- is_pii: true unless category is "Not PII"
- Do not include explanations outside the JSON array"""

        message = self._client.messages.create(
            model=_MODEL,
            max_tokens=1024,
            messages=[{"role": "user", "content": prompt}],
        )
        raw = message.content[0].text.strip()

        # Extract JSON array from response (Claude may wrap in markdown)
        json_match = re.search(r"\[.*\]", raw, re.DOTALL)
        if not json_match:
            raise ValueError(f"Could not parse JSON from Claude response: {raw[:200]}")

        parsed = json.loads(json_match.group(0))

        results = []
        for i, finding in enumerate(findings):
            # Find the matching result by index (1-based in prompt)
            match = next((p for p in parsed if p.get("index") == i + 1), None)
            if match:
                category = match.get("category", "Not PII")
                confidence = float(match.get("confidence", 0.7))
                is_pii = bool(match.get("is_pii", category != "Not PII"))
                results.append({
                    "pii_category": category,
                    "dpdpa_schedule": "Personal Data" if is_pii else "Not Personal Data",
                    "confidence_score": confidence,
                })
            else:
                # No match — keep Layer 1/2 result, cap confidence
                orig_conf = finding.get("confidence_score", 0.0) or 0.0
                results.append({
                    "pii_category": finding.get("pii_category", "Unknown"),
                    "dpdpa_schedule": finding.get("dpdpa_schedule", "Unknown"),
                    "confidence_score": min(orig_conf, 0.64),
                    "classifier": "layer2_fallback",
                })

        return results


# Convenience function for use in execute_scan()
_default_classifier: LLMClassifier | None = None


def get_classifier(redis_client=None) -> LLMClassifier:
    """Return (or create) the module-level LLMClassifier singleton."""
    global _default_classifier
    if _default_classifier is None:
        _default_classifier = LLMClassifier(redis_client=redis_client)
    return _default_classifier
