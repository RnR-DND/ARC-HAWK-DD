from hawk_scanner.internals.system import get_connection
from hawk_scanner.internals.validation_integration import VALIDATOR_MAP
import re


def match_strings(args, content, source='text'):
    redacted = False

    if args and 'connection' in args:
        connections = get_connection(args)
        if 'notify' in connections:
            redacted = connections.get('notify', {}).get('redacted', False)

    from sdk.engine import SharedAnalyzerEngine
    engine = SharedAnalyzerEngine.get_engine()

    results = engine.analyze(
        text=content,
        entities=[
            "IN_IFSC",
            "IN_AADHAAR",
            "IN_PAN",
            "IN_PHONE",
            "EMAIL_ADDRESS",
            "CREDIT_CARD",
            "IN_BANK_ACCOUNT",
            "IN_DRIVING_LICENSE",
            "IN_PASSPORT",
            "IN_VOTER_ID",
            "IN_UPI"
        ],
        language="en"
    )

    print(" PRESIDIO RAW RESULTS:")
    for r in results:
        print("   -", r.entity_type, content[r.start:r.end])

    # -------------------------------
    # STEP 1 — PRIORITY SORT
    # -------------------------------
    PRIORITY = {
        "IN_AADHAAR": 6,
        "CREDIT_CARD": 5,
        "IN_PAN": 4,
        "IN_PASSPORT": 4,
        "IN_DRIVING_LICENSE": 3,
        "IN_UPI": 3,
        "EMAIL_ADDRESS": 3,
        "IN_IFSC": 3,
        "IN_PHONE": 2,
        "IN_BANK_ACCOUNT": 2,
    }

    results = sorted(
        results,
        key=lambda r: (
            r.end - r.start,
            PRIORITY.get(r.entity_type, 0)
        ),
        reverse=True
    )

    def normalize(text):
        return ''.join(c for c in text if c.isalnum())

    # -------------------------------
    # STEP 2 — OVERLAP RESOLUTION
    # -------------------------------
    filtered = []

    for r in results:
        keep = True
        to_remove = []

        for i, f in enumerate(filtered):
            overlap = not (r.end <= f.start or r.start >= f.end)

            if overlap:
                if r.entity_type == f.entity_type:
                    r_pri = PRIORITY.get(r.entity_type, 0)
                    f_pri = PRIORITY.get(f.entity_type, 0)

                    if r.entity_type == "CREDIT_CARD" and f.entity_type == "IN_BANK_ACCOUNT":
                        to_remove.append(i)
                        continue

                    if r.entity_type == "IN_BANK_ACCOUNT" and f.entity_type == "CREDIT_CARD":
                        keep = False
                        break

                    if r_pri > f_pri:
                        to_remove.append(i)
                    else:
                        keep = False
                        break
                else:
                    continue

        if keep:
            for idx in reversed(to_remove):
                filtered.pop(idx)
            filtered.append(r)

    results = filtered

    # -------------------------------
    # STEP 3 — DEDUP (NORMALIZED)
    # -------------------------------
    seen = set()
    deduped = []

    for r in results:
        value = content[r.start:r.end].strip()

        # normalize numeric-heavy entities
        if any(c.isdigit() for c in value):
            normalized = re.sub(r'\D', '', value)
        else:
            normalized = value.lower()

        key = (r.entity_type, normalized)

        if key in seen:
            continue

        seen.add(key)
        deduped.append(r)

    results = deduped

    # -------------------------------
    # STEP 4 — VALIDATION + FILTERING
    # -------------------------------
    findings = []
    seen_findings = set()

    for r in results:
        raw_text = content[r.start:r.end]
        normalized_text = raw_text.strip()
        context = content[max(0, r.start-30):r.end+30].lower()

        # -------------------------------
        # STRICT PHONE FILTER (CRITICAL)
        # -------------------------------
        if r.entity_type == "IN_PHONE":
            clean_phone = re.sub(r"\D", "", normalized_text)

            # strict length
            if len(clean_phone) != 10:
                continue

            # strict prefix
            if clean_phone[0] not in "6789":
                continue

            # 🚨 CRITICAL: reject if part of longer number in original text
            if re.search(rf"\d{clean_phone}\d", content):
                continue

        print("🔥 RAW DETECTION:", r.entity_type, normalized_text)

        # -------------------------------
        # BANK vs CREDIT CARD FIX
        # -------------------------------

        if r.entity_type == "IN_BANK_ACCOUNT":
            clean = re.sub(r'\D', '', normalized_text)

            if len(clean) == 10 and clean[0] in "6789":
                continue

            from sdk.validators.luhn import validate_credit_card
            if validate_credit_card(clean):
                r.entity_type = "CREDIT_CARD"

        if r.entity_type == "CREDIT_CARD":
            normalized_text = re.sub(r"[^\d]", "", normalized_text)

        if r.entity_type == "IN_AADHAAR":
            clean = re.sub(r'\D', '', normalized_text)

            if len(clean) != 12:
                continue

        # -------------------------------
        # VALIDATION
        # -------------------------------

        validator = VALIDATOR_MAP.get(r.entity_type)

        is_valid = True
        if validator:
            is_valid = validator(normalized_text)
            print("VALIDATOR DEBUG:", r.entity_type, "→", is_valid)

        if not is_valid:
            continue

        # -------------------------------
        # FINAL DEDUP (STRONG)
        # -------------------------------

        clean = re.sub(r'\D', '', normalized_text) if any(c.isdigit() for c in normalized_text) else normalized_text.lower()

        key = (r.entity_type, clean, r.start, r.end)

        if key in seen_findings:
            continue

        seen_findings.add(key)

        findings.append({
            "pattern_name": r.entity_type,
            "matches": [normalized_text],
            "confidence_score": r.score,
        })

    print("🔥 FINAL FINDINGS:", findings)

    return findings
