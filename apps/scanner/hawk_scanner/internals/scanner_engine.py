import re
from typing import List, Dict, Any

try:
    import spacy
    # Load small english model if available, else warn
    try:
        nlp = spacy.load("en_core_web_sm")
    except OSError:
        nlp = None
except ImportError:
    nlp = None

try:
    from blingfire import text_to_sentences
except ImportError:
    text_to_sentences = None

from hawk_scanner.internals import entropy
from hawk_scanner.internals import code_analyzer
from hawk_scanner.internals import validation_integration


class ContextAwareScanner:
    def __init__(self, debug=False):
        self.debug = debug
        self.min_entropy_threshold = 3.0  # Slightly lower than 3.5 to be safe

    def scan(self, content: str, patterns: Dict[str, str], source: str = 'text') -> List[Dict[str, Any]]:
        """
        Scans content using context-aware logic, with strict chunking/routing
        for unstructured sources using Blingfire or spaCy when available.
        """
        findings = []

        # Unstructured Chunking / Routing Logic
        # For unstructured sources like fs, s3, slack, we use advanced chunking
        if source in ['fs', 's3', 'slack', 'gdrive']:
            if text_to_sentences is not None:
                # Blingfire for ultra-fast boundary detection
                raw_sentences = text_to_sentences(content).split('\n')
                lines = [s.strip() for s in raw_sentences if s.strip()]
            elif nlp is not None:
                # spaCy as an intelligent fallback chunker
                # Process in blocks if content is huge to avoid memory issues
                if len(content) > 1000000:
                    lines = content.splitlines()  # Fallback for extremely large single blocks
                else:
                    doc = nlp(content)
                    lines = [sent.text.strip()
                             for sent in doc.sents if sent.text.strip()]
            else:
                lines = content.splitlines()
        else:
            # Structured sources (Postgres, MongoDB, Redis, etc.) or fallback
            lines = content.splitlines()

        for pattern_name, pattern_regex in patterns.items():
            compiled_regex = re.compile(pattern_regex, re.IGNORECASE)

            # Iterate line by line for context
            for line_idx, line in enumerate(lines):
                # Use finditer to get full match objects (avoids capturing group issues)
                matches_iter = compiled_regex.finditer(line)

                # Analyze Code Context once per line
                context = code_analyzer.analyze_line_context(line)

                for match_obj in matches_iter:
                    # Get the full match string
                    match_text = match_obj.group(0)

                    # Calculate Confidence Score
                    score, reasons = self._calculate_confidence(
                        match_text, pattern_name, context)

                    # Validate using SDK (Strict Validation Phase 1 logic)
                    is_valid_format, method = validation_integration.validate_match(
                        match_text, pattern_name)

                    if is_valid_format:
                        # If SDK says it's valid (checksum ok), boost score significantly
                        score = 100
                        reasons.append(f"SDK Validation Passed ({method})")
                    elif method == 'no_validator':
                        # No validator exists, rely on heuristic score
                        pass
                    elif method == 'scope_rejected':
                        # See previous notes
                        pass
                    else:
                        # Validator existed but failed (checksum fail)
                        score = 0
                        reasons.append("SDK Validation Failed")

                    # Final Decision
                    if score >= 50:
                        findings.append({
                            'data_source': source,
                            'pattern_name': pattern_name,
                            'matches': [match_text],
                            'sample_text': line[:100],  # Line context
                            'line_number': line_idx + 1,
                            'confidence_score': score,
                            'confidence_reasons': reasons,
                            'validation_method': method
                        })

        return self._deduplicate_findings(findings)

    def _calculate_confidence(self, match: str, pattern_name: str, context: Dict) -> tuple[int, List[str]]:
        score = 50  # Base score
        reasons = []

        # 1. Entropy Check
        ent = entropy.calculate_shannon_entropy(match)
        if ent > self.min_entropy_threshold:
            score += 20
            reasons.append(f"High Entropy ({ent:.2f})")
        else:
            # Low entropy for secrets is bad (e.g. "password"), but fine for PII like Phone numbers
            # If pattern is a Key/Secret, punish low entropy
            if "key" in pattern_name.lower() or "secret" in pattern_name.lower():
                score -= 20
                reasons.append(f"Low Entropy ({ent:.2f})")

        # 2. Context Check
        if context['is_assignment']:
            if context['has_sensitive_keyword']:
                score += 30
                reasons.append(
                    f"Sensitive Variable Assignment ({context['variable_name']})")
            else:
                score += 10
                reasons.append("Variable Assignment")

        if context['is_comment']:
            score -= 30  # Reduce confidence for commented out secrets
            reasons.append("In Comment")

        # 3. Test Data Check
        if "example" in match.lower() or "test" in match.lower() or "12345" in match:
            score = 0
            reasons.append("Test Data Value")

        return max(0, min(100, score)), reasons

    def _deduplicate_findings(self, findings):
        # ... Simple deduplication logic ...
        # For now return as is or minimal dedup
        return findings
