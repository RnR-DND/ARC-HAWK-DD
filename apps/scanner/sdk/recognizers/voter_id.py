"""
Indian Voter ID Recognizer
===========================
Detects Indian Voter IDs (EPIC - Electors Photo Identity Card).

Format: 3 letters + 7 digits (e.g., ABC1234567)
"""

from typing import Optional
from presidio_analyzer import Pattern, PatternRecognizer
from sdk.validators.voter_id import validate_voter_id
import re

class VoterIDRecognizer(PatternRecognizer):
    """Custom recognizer for Indian Voter IDs."""
    
    PATTERNS = [
        Pattern(
            name="Voter ID (AAA9999999)",
            regex=r"(?i)(?:^|[^A-Z0-9])([A-Z]{3}[\s\-]?[0-9]{7})(?![A-Z0-9])",
            score=0.5
        ),
    ]
    
    CONTEXT = [
        "voter",
        "voter id",
        "epic",
        "election card",
        "electoral",
        "voter card",
    ]
    
    def __init__(self):
        super().__init__(
            supported_entity="IN_VOTER_ID",
            name="Voter ID Recognizer",
            supported_language="en",
            patterns=self.PATTERNS,
            context=self.CONTEXT
        )
    
    def validate_result(self, pattern_text: str) -> Optional[bool]:
        """Validate Voter ID format using strict validator."""
        normalized = re.sub(r'[\s\-/]', '', pattern_text.upper())
        # Use the actual voter ID validator
        return validate_voter_id(normalized)


if __name__ == "__main__":
    recognizer = VoterIDRecognizer()
    print(f"Voter ID Recognizer created: {recognizer.name}")
