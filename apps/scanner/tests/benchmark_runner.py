"""
DPDPA Benchmark Suite Runner

Runs the scanner's classification pipeline against labelled test fixtures and
reports precision, recall, and F1 per DPDPA category.

Usage:
    python tests/benchmark_runner.py [--categories aadhaar pan gstin ...] [--tier all|true_positives|...]

CI gate: any category with F1 drop > 2% from baseline fails the run.
Baseline is stored in testdata/benchmarks/baseline_f1.json.
"""

import argparse
import json
import os
import sys
from pathlib import Path
from typing import Any

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------
BENCHMARK_DIR = Path(__file__).parent.parent / "testdata" / "benchmarks"
BASELINE_FILE = BENCHMARK_DIR / "baseline_f1.json"
F1_REGRESSION_THRESHOLD = 0.02  # 2% drop triggers failure

ALL_CATEGORIES = [
    "aadhaar", "pan", "gstin", "ifsc", "upi", "phone",
    "email", "passport", "voter_id", "dob", "name",
    "health_record", "financial_id",
]
ALL_TIERS = ["true_positives", "true_negatives", "edge_cases"]


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def load_fixtures(category: str, tier: str) -> list[dict]:
    path = BENCHMARK_DIR / category / f"{tier}.jsonl"
    if not path.exists():
        return []
    records = []
    with open(path) as f:
        for line in f:
            line = line.strip()
            if line:
                records.append(json.loads(line))
    return records


def run_classifier(record: dict) -> dict:
    """
    Run the scanner's classification pipeline against a single fixture record.
    Returns a dict with 'predicted_category', 'confidence', 'layer'.

    Uses sdk.validators directly (sdk.recognizers has been removed).
    """
    import re
    from sdk.validators.verhoeff import validate_aadhaar
    from sdk.validators.pan import validate_pan
    from sdk.validators.ifsc import validate_ifsc
    from sdk.validators.upi import validate_upi

    value = record.get("raw_value", "")
    expected = record.get("expected_category", "")

    PATTERNS = {
        "aadhaar": (r"^[2-9]\d{11}$", lambda v: validate_aadhaar(v.replace(" ", "").replace("-", "")), "Government ID"),
        "pan": (r"^[A-Z]{5}[0-9]{4}[A-Z]$", validate_pan, "Financial ID"),
        "gstin": (r"^\d{2}[A-Z]{5}\d{4}[A-Z][1-9A-Z]Z[0-9A-Z]$", None, "Government ID"),
        "ifsc": (r"^[A-Z]{4}0[A-Z0-9]{6}$", validate_ifsc, "Financial ID"),
        "upi": (r"^[a-zA-Z0-9._-]+@[a-zA-Z0-9._-]+$", validate_upi, "Contact Information"),
        "email": (r"[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}", None, "Contact Information"),
        "phone": (r"(\+91[\s-]?)?[6-9]\d{9}", None, "Contact Information"),
        "passport": (r"^[A-Z][1-9][0-9]{7}$", None, "Government ID"),
        "voter_id": (r"^[A-Z]{3}[0-9]{7}$", None, "Government ID"),
    }

    for cat, (pattern, validator_fn, category) in PATTERNS.items():
        if re.search(pattern, value.strip()):
            if validator_fn is not None:
                try:
                    clean = value.replace(" ", "").replace("-", "")
                    if not validator_fn(clean):
                        continue
                except Exception:
                    continue
            return {"predicted_category": category, "confidence": 0.9, "layer": "rule_based"}

    return {"predicted_category": "Not PII", "confidence": 0.3, "layer": "rule_based"}


def compute_metrics(tp: int, fp: int, fn: int) -> dict:
    precision = tp / (tp + fp) if (tp + fp) > 0 else 0.0
    recall = tp / (tp + fn) if (tp + fn) > 0 else 0.0
    f1 = (2 * precision * recall / (precision + recall)) if (precision + recall) > 0 else 0.0
    return {"precision": round(precision, 4), "recall": round(recall, 4), "f1": round(f1, 4)}


# ---------------------------------------------------------------------------
# Runner
# ---------------------------------------------------------------------------

def run_benchmark(categories: list[str], tiers: list[str], verbose: bool = False) -> dict:
    results: dict[str, Any] = {}
    overall_tp = overall_fp = overall_fn = 0

    for category in categories:
        tp = fp = fn = 0
        for tier in tiers:
            fixtures = load_fixtures(category, tier)
            for record in fixtures:
                expected = record.get("expected_category", "Not PII")
                is_positive = expected != "Not PII"

                prediction = run_classifier(record)
                predicted = prediction.get("predicted_category", "Not PII")
                predicted_positive = predicted != "Not PII"

                if is_positive and predicted_positive:
                    tp += 1
                elif not is_positive and predicted_positive:
                    fp += 1
                elif is_positive and not predicted_positive:
                    fn += 1

                if verbose:
                    match = "✓" if (is_positive == predicted_positive) else "✗"
                    print(f"  {match} [{category}/{tier}] {record['id']}: "
                          f"expected={expected}, predicted={predicted}")

        metrics = compute_metrics(tp, fp, fn)
        results[category] = {**metrics, "tp": tp, "fp": fp, "fn": fn}
        overall_tp += tp
        overall_fp += fp
        overall_fn += fn

    results["_overall"] = compute_metrics(overall_tp, overall_fp, overall_fn)
    return results


def check_regression(results: dict) -> list[str]:
    """Return list of categories that regressed > 2% F1 from baseline."""
    if not BASELINE_FILE.exists():
        print(f"No baseline found at {BASELINE_FILE} — writing current results as baseline")
        with open(BASELINE_FILE, "w") as f:
            json.dump({k: v["f1"] for k, v in results.items()}, f, indent=2)
        return []

    with open(BASELINE_FILE) as f:
        baseline = json.load(f)

    regressions = []
    for category, metrics in results.items():
        baseline_f1 = baseline.get(category, 0.0)
        current_f1 = metrics.get("f1", 0.0)
        drop = baseline_f1 - current_f1
        if drop > F1_REGRESSION_THRESHOLD:
            regressions.append(
                f"{category}: F1 dropped {drop:.3f} ({baseline_f1:.3f} → {current_f1:.3f})"
            )
    return regressions


def print_report(results: dict) -> None:
    print("\n" + "=" * 70)
    print("DPDPA Benchmark Suite — Classification Accuracy Report")
    print("=" * 70)
    print(f"{'Category':<20} {'Precision':>10} {'Recall':>10} {'F1':>10}  TP  FP  FN")
    print("-" * 70)
    for category, m in sorted(results.items()):
        if category.startswith("_"):
            continue
        f1_flag = " ⚠" if m["f1"] < 0.90 else ""
        print(
            f"{category:<20} {m['precision']:>10.4f} {m['recall']:>10.4f} "
            f"{m['f1']:>10.4f}{f1_flag}  {m['tp']:>3} {m['fp']:>3} {m['fn']:>3}"
        )
    print("-" * 70)
    ov = results.get("_overall", {})
    print(
        f"{'OVERALL':<20} {ov.get('precision', 0):>10.4f} {ov.get('recall', 0):>10.4f} "
        f"{ov.get('f1', 0):>10.4f}"
    )
    print("=" * 70)

    below_target = [c for c, m in results.items() if not c.startswith("_") and m["f1"] < 0.90]
    if below_target:
        print(f"\n⚠  Categories below 90% F1 target: {', '.join(below_target)}")
    else:
        print("\n✓  All categories at or above 90% F1 target")


def main():
    parser = argparse.ArgumentParser(description="DPDPA benchmark suite runner")
    parser.add_argument("--categories", nargs="+", default=ALL_CATEGORIES)
    parser.add_argument("--tiers", nargs="+", default=ALL_TIERS)
    parser.add_argument("--verbose", action="store_true")
    parser.add_argument("--update-baseline", action="store_true",
                        help="Write current results as the new baseline")
    args = parser.parse_args()

    results = run_benchmark(args.categories, args.tiers, verbose=args.verbose)
    print_report(results)

    if args.update_baseline:
        with open(BASELINE_FILE, "w") as f:
            json.dump({k: v["f1"] for k, v in results.items()}, f, indent=2)
        print(f"\nBaseline updated at {BASELINE_FILE}")
        return 0

    regressions = check_regression(results)
    if regressions:
        print(f"\n✗  F1 REGRESSION DETECTED (threshold: {F1_REGRESSION_THRESHOLD*100:.0f}%):")
        for r in regressions:
            print(f"   {r}")
        return 1

    below = [c for c, m in results.items() if not c.startswith("_") and m["f1"] < 0.90]
    if below:
        print(f"\n✗  Categories below 90% F1 target: {', '.join(below)}")
        return 1

    print("\n✓  All checks passed")
    return 0


if __name__ == "__main__":
    sys.exit(main())
