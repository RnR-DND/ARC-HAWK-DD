"""
Context-Aware Validator - Enhanced PII Detection
=================================================
Reduces false positives by detecting test data patterns and analyzing context.

Features:
- Test data pattern detection (9999999999, test@test.com, etc.)
- Context keyword extraction
- Confidence adjustment based on context
- Exclusion list support
"""

import re
from typing import List, Set, Optional, Dict
from dataclasses import dataclass


@dataclass
class ValidationContext:
    """Context information for validation"""
    value: str
    pii_type: str
    surrounding_text: str
    keywords: List[str]
    base_confidence: float


# ==================================================================================
# TEST DATA PATTERNS - Common test/dummy data that should be filtered
# ==================================================================================

TEST_DATA_PATTERNS = {
    "IN_PHONE": [
        r"^9{10}$",           # 9999999999
        r"^0{10}$",           # 0000000000
        r"^1{10}$",           # 1111111111
        r"^1234567890$",      # Sequential
        r"^0987654321$",      # Reverse sequential
    ],
    "EMAIL_ADDRESS": [
        r"^test@test\.com$",
        r"^test@example\.com$",
        r"^dummy@dummy\.com$",
        r"^sample@sample\.com$",
        r"^foo@bar\.com$",
        r"^noreply@.*",
        r"^no-reply@.*",
    ],
    "IN_AADHAAR": [
        r"^1{12}$",           # 111111111111
        r"^0{12}$",           # 000000000000
        r"^9{12}$",           # 999999999999
        r"^123456789012$",    # Sequential
    ],
    "IN_PAN": [
        r"^AAAAA0000A$",      # All A's with zeros
        r"^ZZZZZ9999Z$",      # All Z's with nines
        r"^TEST[A-Z]0000[A-Z]$",  # Contains TEST
    ],
    "CREDIT_CARD": [
        r"^1{16}$",           # 1111111111111111
        r"^0{16}$",           # 0000000000000000
        r"^1234567890123456$", # Sequential
    ],
    "IN_BANK_ACCOUNT": [
        r"^0{10,}$",          # All zeros
        r"^1{10,}$",          # All ones
        r"^123456789012345$", # Sequential
    ],
}


# ==================================================================================
# CONTEXT KEYWORDS - Keywords that indicate test/production environment
# ==================================================================================

TEST_KEYWORDS = {
    "test", "testing", "dummy", "sample", "example", "fake", "mock",
    "demo", "sandbox", "dev", "development", "staging", "qa"
}

PRODUCTION_KEYWORDS = {
    "production", "prod", "live", "customer", "client", "user",
    "employee", "patient", "member", "account", "real"
}

NEGATIVE_KEYWORDS = {
    "invalid", "incorrect", "wrong", "error", "failed", "rejected"
}


class ContextValidator:
    """
    Context-aware validator that reduces false positives.
    """
    
    def __init__(self, exclusion_list: Optional[List[str]] = None):
        """
        Initialize context validator.
        
        Args:
            exclusion_list: Optional list of values to exclude from detection
        """
        self.exclusion_list: Set[str] = set(exclusion_list or [])
        self._compile_patterns()
    
    def _compile_patterns(self):
        """Compile regex patterns for performance"""
        self.compiled_patterns: Dict[str, List[re.Pattern]] = {}
        for pii_type, patterns in TEST_DATA_PATTERNS.items():
            self.compiled_patterns[pii_type] = [
                re.compile(pattern, re.IGNORECASE) for pattern in patterns
            ]
    
    def is_test_data(self, value: str, pii_type: str) -> bool:
        """
        Check if a value matches common test data patterns.
        
        Args:
            value: PII value to check
            pii_type: PII type (e.g., "IN_PHONE", "EMAIL_ADDRESS")
            
        Returns:
            True if value appears to be test data
        """
        # Clean value for pattern matching
        cleaned = value.strip().replace(" ", "").replace("-", "")
        
        # Check against test data patterns for this PII type
        patterns = self.compiled_patterns.get(pii_type, [])
        for pattern in patterns:
            if pattern.match(cleaned):
                return True
        
        # Check exclusion list
        if value in self.exclusion_list or cleaned in self.exclusion_list:
            return True
        
        return False
    
    def extract_context_keywords(
        self,
        text: str,
        start: int,
        end: int,
        window_size: int = 100
    ) -> List[str]:
        """
        Extract keywords from surrounding context.
        
        Args:
            text: Full text
            start: Start position of PII match
            end: End position of PII match
            window_size: Number of characters to include on each side
            
        Returns:
            List of keywords found in context
        """
        # Extract context window
        context_start = max(0, start - window_size)
        context_end = min(len(text), end + window_size)
        context = text[context_start:context_end].lower()
        
        # Extract words
        words = re.findall(r'\b\w+\b', context)
        
        # Filter to relevant keywords
        all_keywords = TEST_KEYWORDS | PRODUCTION_KEYWORDS | NEGATIVE_KEYWORDS
        found_keywords = [word for word in words if word in all_keywords]
        
        return found_keywords
    
    def adjust_confidence_by_context(
        self,
        base_confidence: float,
        keywords: List[str]
    ) -> float:
        """
        Adjust confidence score based on context keywords.
        
        Args:
            base_confidence: Original confidence score (0.0-1.0)
            keywords: Keywords found in context
            
        Returns:
            Adjusted confidence score (0.0-1.0)
        """
        confidence = base_confidence
        
        # Count keyword types
        test_count = sum(1 for kw in keywords if kw in TEST_KEYWORDS)
        prod_count = sum(1 for kw in keywords if kw in PRODUCTION_KEYWORDS)
        negative_count = sum(1 for kw in keywords if kw in NEGATIVE_KEYWORDS)
        
        # Adjust confidence
        if test_count > 0:
            # Lower confidence if test keywords present
            confidence *= (1.0 - (test_count * 0.15))  # -15% per test keyword
        
        if prod_count > 0:
            # Increase confidence if production keywords present
            confidence *= (1.0 + (prod_count * 0.05))  # +5% per prod keyword
        
        if negative_count > 0:
            # Significantly lower confidence if negative keywords present
            confidence *= (1.0 - (negative_count * 0.25))  # -25% per negative keyword
        
        # Clamp to valid range
        return max(0.0, min(1.0, confidence))
    
    def validate_with_context(
        self,
        value: str,
        pii_type: str,
        text: str,
        start: int,
        end: int,
        base_confidence: float = 0.95
    ) -> tuple[bool, float, str]:
        """
        Validate a PII value with context awareness.
        
        Args:
            value: PII value to validate
            pii_type: PII type
            text: Full text containing the PII
            start: Start position of match
            end: End position of match
            base_confidence: Base confidence from mathematical validation
            
        Returns:
            Tuple of (is_valid, adjusted_confidence, rejection_reason)
        """
        # Check if it's test data
        if self.is_test_data(value, pii_type):
            return False, 0.0, f"Rejected: Test data pattern detected ({value})"
        
        # Extract context keywords
        keywords = self.extract_context_keywords(text, start, end)
        
        # Adjust confidence based on context
        adjusted_confidence = self.adjust_confidence_by_context(base_confidence, keywords)
        
        # Reject if confidence drops too low
        if adjusted_confidence < 0.5:
            keyword_str = ", ".join(keywords) if keywords else "none"
            return False, adjusted_confidence, f"Rejected: Low confidence after context analysis (keywords: {keyword_str})"
        
        return True, adjusted_confidence, "Valid"
    
    def add_to_exclusion_list(self, values: List[str]):
        """Add values to exclusion list"""
        self.exclusion_list.update(values)
    
    def get_statistics(self) -> Dict[str, int]:
        """Get validation statistics"""
        return {
            "test_patterns_count": sum(len(patterns) for patterns in TEST_DATA_PATTERNS.values()),
            "exclusion_list_size": len(self.exclusion_list),
            "test_keywords_count": len(TEST_KEYWORDS),
            "production_keywords_count": len(PRODUCTION_KEYWORDS),
        }


# ==================================================================================
# UTILITY FUNCTIONS
# ==================================================================================

def is_sequential_number(value: str) -> bool:
    """
    Check if a number is sequential (e.g., 123456789).
    
    Args:
        value: Numeric string
        
    Returns:
        True if sequential
    """
    if not value.isdigit() or len(value) < 4:
        return False
    
    # Check ascending sequence
    is_ascending = all(
        int(value[i+1]) == int(value[i]) + 1
        for i in range(len(value) - 1)
    )
    
    # Check descending sequence
    is_descending = all(
        int(value[i+1]) == int(value[i]) - 1
        for i in range(len(value) - 1)
    )
    
    return is_ascending or is_descending


def is_repeating_pattern(value: str) -> bool:
    """
    Check if a value is a repeating pattern (e.g., 111111, AAAA).
    
    Args:
        value: String to check
        
    Returns:
        True if repeating pattern
    """
    if len(value) < 3:
        return False
    
    # Check if all characters are the same
    return len(set(value)) == 1


# ==================================================================================
# TESTING
# ==================================================================================

if __name__ == "__main__":
    print("=== Context Validator Test ===\n")
    
    validator = ContextValidator()
    
    # Test 1: Test data detection
    print("Test 1: Test Data Detection")
    test_cases = [
        ("9999999999", "IN_PHONE", True),
        ("9876543210", "IN_PHONE", False),
        ("test@test.com", "EMAIL_ADDRESS", True),
        ("john@company.com", "EMAIL_ADDRESS", False),
        ("111111111111", "IN_AADHAAR", True),
        ("234567890124", "IN_AADHAAR", False),
    ]
    
    for value, pii_type, expected_test_data in test_cases:
        is_test = validator.is_test_data(value, pii_type)
        status = "✓" if is_test == expected_test_data else "✗"
        print(f"  {status} {value} ({pii_type}): is_test={is_test} (expected={expected_test_data})")
    
    # Test 2: Context keyword extraction
    print("\nTest 2: Context Keyword Extraction")
    text = "This is a test customer with phone 9876543210 in production"
    keywords = validator.extract_context_keywords(text, 38, 48)
    print(f"  Text: {text}")
    print(f"  Keywords: {keywords}")
    
    # Test 3: Confidence adjustment
    print("\nTest 3: Confidence Adjustment")
    test_keywords = ["test", "dummy"]
    prod_keywords = ["production", "customer"]
    
    base_conf = 0.95
    test_conf = validator.adjust_confidence_by_context(base_conf, test_keywords)
    prod_conf = validator.adjust_confidence_by_context(base_conf, prod_keywords)
    
    print(f"  Base confidence: {base_conf}")
    print(f"  With test keywords {test_keywords}: {test_conf:.2f}")
    print(f"  With prod keywords {prod_keywords}: {prod_conf:.2f}")
    
    # Test 4: Full validation with context
    print("\nTest 4: Full Validation with Context")
    test_text = "Test data: phone is 9876543210 for testing purposes"
    is_valid, conf, reason = validator.validate_with_context(
        "9876543210", "IN_PHONE", test_text, 24, 34
    )
    print(f"  Text: {test_text}")
    print(f"  Valid: {is_valid}, Confidence: {conf:.2f}")
    print(f"  Reason: {reason}")
    
    # Statistics
    print("\nValidator Statistics:")
    stats = validator.get_statistics()
    for key, value in stats.items():
        print(f"  {key}: {value}")
