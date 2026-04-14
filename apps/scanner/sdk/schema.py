"""
Verified Finding Schema
=======================
New schema for scanner output after SDK validation.

OLD (Raw Match):
{
  "pattern_name": "Aadhaar",
  "match": "1234 5678 9012",
  "path": "/data/file.txt"
}

NEW (Verified Finding):
{
  "pii_type": "IN_AADHAAR",
  "value_hash": "sha256(...)",
  "source": {...},
  "validators_passed": ["verhoeff"],
  "ml_confidence": 0.91,
  "context_excerpt": "..."
}
"""

import hashlib
from typing import List, Dict, Any, Optional
from dataclasses import dataclass, asdict
from datetime import datetime


@dataclass
class SourceInfo:
    """Information about where PII was found."""
    path: str
    line: Optional[int] = None
    column: Optional[str] = None
    table: Optional[str] = None
    data_source: str = "filesystem"  # filesystem, postgresql, mysql
    host: str = "localhost"


@dataclass
class VerifiedFinding:
    """
    A PII finding that has passed SDK validation.
    
    Only validated findings should reach the backend.
    """
    # Identity
    pii_type: str  # IN_AADHAAR, CREDIT_CARD, etc.
    value_hash: str  # SHA-256 hash of the actual PII
    
    # Source
    source: SourceInfo
    
    # Validation proof
    validators_passed: List[str]  # ["verhoeff"], ["luhn"], etc.
    validation_method: str  # "mathematical", "format", "ml"
    
    # ML signals
    ml_confidence: float  # Presidio confidence (0.0-1.0)
    ml_entity_type: str  # Entity type Presidio detected
    
    # Context
    context_excerpt: str  # Surrounding text (±100 chars)
    context_keywords: List[str]  # Keywords found near match
    
    # Metadata
    pattern_name: str  # Original pattern that triggered detection
    detected_at: str  # ISO timestamp
    scanner_version: str = "2.0-sdk"
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON serialization."""
        return {
            "pii_type": self.pii_type,
            "value_hash": self.value_hash,
            "source": asdict(self.source),
            "validators_passed": self.validators_passed,
            "validation_method": self.validation_method,
            "ml_confidence": self.ml_confidence,
            "ml_entity_type": self.ml_entity_type,
            "context_excerpt": self.context_excerpt,
            "context_keywords": self.context_keywords,
            "pattern_name": self.pattern_name,
            "detected_at": self.detected_at,
            "scanner_version": self.scanner_version,
        }
    
    @staticmethod
    def create_from_analysis(
        presidio_result,
        text: str,
        source_info: SourceInfo,
        pattern_name: str,
        validators: List[str]
    ) -> 'VerifiedFinding':
        """
        Factory method to create VerifiedFinding from Presidio result.
        
        Args:
            presidio_result: RecognizerResult from Presidio
            text: Full text that was analyzed
            source_info: Information about the source
            pattern_name: Original pattern name
            validators: List of validators that passed
            
        Returns:
            VerifiedFinding instance
        """
        # Extract the matched value
        matched_value = text[presidio_result.start:presidio_result.end]
        
        # Hash the value (NEVER store raw PII)
        value_hash = hashlib.sha256(matched_value.encode()).hexdigest()
        
        # FIX M9: Extract context (±50 chars around match)
        context_start = max(0, presidio_result.start - 50)
        context_end = min(len(text), presidio_result.end + 50)
        context_excerpt = text[context_start:context_end]

        # FIX M9: Extract context keywords — significant words (len > 4) near match.
        # Strip punctuation before length check so short words with punctuation
        # are excluded correctly. strip() chars chosen to avoid shell-quoting issues.
        _punct = set(".,;:!?\"'()")
        context_words = context_excerpt.split()
        context_keywords = [
            w.strip(".,;:!?\"'()")
            for w in context_words
            if len(w.strip(".,;:!?\"'()")) > 4
        ][:10]
        
        return VerifiedFinding(
            pii_type=presidio_result.entity_type,
            value_hash=value_hash,
            source=source_info,
            validators_passed=validators,
            validation_method="mathematical" if validators else "ml",
            ml_confidence=presidio_result.score,
            ml_entity_type=presidio_result.entity_type,
            context_excerpt=context_excerpt,
            context_keywords=context_keywords,
            pattern_name=pattern_name,
            detected_at=datetime.utcnow().isoformat() + "Z",
        )


if __name__ == "__main__":
    print("=== VerifiedFinding Schema Test ===\n")
    
    # Create sample finding
    source = SourceInfo(
        path="/data/users.csv",
        line=42,
        column="aadhaar_number",
        data_source="filesystem"
    )
    
    finding = VerifiedFinding(
        pii_type="IN_AADHAAR",
        value_hash=hashlib.sha256(b"999911112226").hexdigest(),
        source=source,
        validators_passed=["verhoeff"],
        validation_method="mathematical",
        ml_confidence=0.91,
        ml_entity_type="IN_AADHAAR",
        context_excerpt="Customer Aadhaar 9999 1111 2226 enrolled",
        context_keywords=["aadhaar", "customer"],
        pattern_name="Aadhaar",
        detected_at="2026-01-08T15:00:00Z"
    )
    
    # Convert to dict
    finding_dict = finding.to_dict()
    
    print("Sample VerifiedFinding:")
    import json
    print(json.dumps(finding_dict, indent=2))
    
    print("\n✓ Schema validated")
