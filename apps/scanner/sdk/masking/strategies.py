"""
Masking Strategies - Universal PII Masking
==========================================
Defines different strategies for masking PII data.

Strategies:
- REDACT: Replace with [REDACTED]
- PARTIAL: Show first/last characters (e.g., XXXX-XXXX-1234)
- TOKENIZE: Replace with reversible token
- FPE: Format-Preserving Encryption (maintains format/length)
"""

from abc import ABC, abstractmethod
from typing import Optional
import hashlib
import re


class MaskingStrategy(ABC):
    """Base class for masking strategies"""
    
    @abstractmethod
    def mask(self, value: str, pii_type: str) -> str:
        """
        Apply masking to a PII value.
        
        Args:
            value: PII value to mask
            pii_type: Type of PII (e.g., "IN_AADHAAR", "CREDIT_CARD")
            
        Returns:
            Masked value
        """
        pass
    
    @abstractmethod
    def get_name(self) -> str:
        """Get strategy name"""
        pass


class RedactStrategy(MaskingStrategy):
    """Replace entire value with [REDACTED]"""
    
    def mask(self, value: str, pii_type: str) -> str:
        return "[REDACTED]"
    
    def get_name(self) -> str:
        return "REDACT"


class PartialMaskStrategy(MaskingStrategy):
    """Show first/last characters, mask the middle"""
    
    def get_name(self) -> str:
        return "PARTIAL"
    
    def mask(self, value: str, pii_type: str) -> str:
        # Remove whitespace and special characters for processing
        cleaned = value.replace(" ", "").replace("-", "").replace("_", "")
        length = len(cleaned)
        
        # For very short values, just redact
        if length <= 4:
            return "[REDACTED]"
        
        # Different strategies based on PII type
        if "AADHAAR" in pii_type.upper():
            # Aadhaar: Show last 4 digits (e.g., XXXX-XXXX-1234)
            if length >= 12:
                return "XXXX-XXXX-" + cleaned[-4:]
            return "XXXX-" + cleaned[-4:]
        
        elif "PAN" in pii_type.upper():
            # PAN: Show first 3 and last 4 (e.g., ABC****1234)
            if length >= 10:
                return cleaned[:3] + "****" + cleaned[-4:]
            return cleaned[:2] + "****" + cleaned[-2:]
        
        elif "PHONE" in pii_type.upper():
            # Phone: Show last 4 digits (e.g., ******1234)
            if length >= 10:
                return "******" + cleaned[-4:]
            return "****" + cleaned[-4:]
        
        elif "EMAIL" in pii_type.upper():
            # Email: Show first 2 chars and domain (e.g., ab****@example.com)
            if "@" in value:
                parts = value.split("@")
                if len(parts) == 2 and len(parts[0]) > 2:
                    return parts[0][:2] + "****@" + parts[1]
                return "****@" + parts[-1]
            return "[REDACTED]"
        
        elif "CARD" in pii_type.upper() or "CREDIT" in pii_type.upper():
            # Credit Card: Show last 4 digits (e.g., ****-****-****-1234)
            if length >= 16:
                return "****-****-****-" + cleaned[-4:]
            return "****-" + cleaned[-4:]
        
        elif "PASSPORT" in pii_type.upper():
            # Passport: Show first char and last 3 (e.g., A****567)
            if length >= 7:
                return cleaned[0] + "****" + cleaned[-3:]
            return cleaned[0] + "****"
        
        elif "VOTER" in pii_type.upper() or "LICENSE" in pii_type.upper():
            # Voter ID / License: Show first 3 and last 3 (e.g., ABC****567)
            if length >= 8:
                return cleaned[:3] + "****" + cleaned[-3:]
            return cleaned[:2] + "****" + cleaned[-2:]
        
        elif "BANK" in pii_type.upper() or "ACCOUNT" in pii_type.upper():
            # Bank Account: Show last 4 digits (e.g., **********1234)
            return "*" * (length - 4) + cleaned[-4:]
        
        elif "UPI" in pii_type.upper():
            # UPI: Show first 2 chars and payment provider (e.g., ab****@paytm)
            if "@" in value:
                parts = value.split("@")
                if len(parts) == 2 and len(parts[0]) > 2:
                    return parts[0][:2] + "****@" + parts[1]
                return "****@" + parts[-1]
            return "[REDACTED]"
        
        elif "IFSC" in pii_type.upper():
            # IFSC: Show first 4 chars (bank code) (e.g., SBIN****234)
            if length >= 11:
                return cleaned[:4] + "****" + cleaned[-3:]
            return cleaned[:4] + "****"
        
        else:
            # Generic: Show first 2 and last 4
            if length > 6:
                return cleaned[:2] + "*" * (length - 6) + cleaned[-4:]
            return cleaned[:1] + "*" * (length - 2) + cleaned[-1:]


class TokenizeStrategy(MaskingStrategy):
    """Replace with consistent, reversible token"""
    
    def __init__(self, secret_key: Optional[str] = None):
        """
        Initialize tokenization strategy.
        
        Args:
            secret_key: Optional secret key for token generation
        """
        self.secret_key = secret_key or "default_secret_key_change_in_production"
    
    def mask(self, value: str, pii_type: str) -> str:
        # Generate deterministic token using HMAC-SHA256
        combined = f"{self.secret_key}:{value}:{pii_type}"
        hash_obj = hashlib.sha256(combined.encode('utf-8'))
        token = hash_obj.hexdigest()[:16].upper()
        
        return f"TOKEN_{token}"
    
    def get_name(self) -> str:
        return "TOKENIZE"


class FPEStrategy(MaskingStrategy):
    """
    Format-Preserving Encryption (simplified version).
    Maintains the format and length of the original value.
    """
    
    def __init__(self, secret_key: Optional[str] = None):
        """
        Initialize FPE strategy.
        
        Args:
            secret_key: Optional secret key for encryption
        """
        self.secret_key = secret_key or "default_fpe_key_change_in_production"
    
    def mask(self, value: str, pii_type: str) -> str:
        """
        Apply format-preserving encryption.
        This is a simplified version - production should use proper FPE library.
        """
        # Determine format
        is_numeric = value.replace(" ", "").replace("-", "").isdigit()
        has_letters = any(c.isalpha() for c in value)
        
        if is_numeric:
            # Numeric FPE: Replace each digit with encrypted digit
            return self._fpe_numeric(value)
        elif has_letters:
            # Alphanumeric FPE: Replace each char with encrypted char
            return self._fpe_alphanumeric(value)
        else:
            # Fallback to tokenization
            return TokenizeStrategy(self.secret_key).mask(value, pii_type)
    
    def _fpe_numeric(self, value: str) -> str:
        """Format-preserving encryption for numeric values"""
        result = []
        for i, char in enumerate(value):
            if char.isdigit():
                # Encrypt digit using position-based key
                key_char = self.secret_key[i % len(self.secret_key)]
                encrypted_digit = (int(char) + ord(key_char)) % 10
                result.append(str(encrypted_digit))
            else:
                # Preserve non-digit characters (spaces, hyphens)
                result.append(char)
        return ''.join(result)
    
    def _fpe_alphanumeric(self, value: str) -> str:
        """Format-preserving encryption for alphanumeric values"""
        result = []
        for i, char in enumerate(value):
            if char.isalnum():
                # Encrypt character using position-based key
                key_char = self.secret_key[i % len(self.secret_key)]
                if char.isdigit():
                    encrypted = (int(char) + ord(key_char)) % 10
                    result.append(str(encrypted))
                elif char.isupper():
                    encrypted = chr((ord(char) - ord('A') + ord(key_char)) % 26 + ord('A'))
                    result.append(encrypted)
                else:
                    encrypted = chr((ord(char) - ord('a') + ord(key_char)) % 26 + ord('a'))
                    result.append(encrypted)
            else:
                # Preserve non-alphanumeric characters
                result.append(char)
        return ''.join(result)
    
    def get_name(self) -> str:
        return "FPE"


# Strategy factory
def get_masking_strategy(strategy_name: str, secret_key: Optional[str] = None) -> MaskingStrategy:
    """
    Get masking strategy by name.
    
    Args:
        strategy_name: Strategy name (REDACT, PARTIAL, TOKENIZE, FPE)
        secret_key: Optional secret key for tokenization/FPE
        
    Returns:
        MaskingStrategy instance
        
    Raises:
        ValueError: If strategy name is invalid
    """
    strategy_name = strategy_name.upper()
    
    if strategy_name == "REDACT":
        return RedactStrategy()
    elif strategy_name == "PARTIAL":
        return PartialMaskStrategy()
    elif strategy_name == "TOKENIZE":
        return TokenizeStrategy(secret_key)
    elif strategy_name == "FPE":
        return FPEStrategy(secret_key)
    else:
        raise ValueError(f"Invalid masking strategy: {strategy_name}. "
                        f"Valid options: REDACT, PARTIAL, TOKENIZE, FPE")


if __name__ == "__main__":
    print("=== Masking Strategies Test ===\n")
    
    # Test data
    test_cases = [
        ("234567890124", "IN_AADHAAR"),
        ("ABCDE1234F", "IN_PAN"),
        ("9876543210", "IN_PHONE"),
        ("john.doe@company.com", "EMAIL_ADDRESS"),
        ("4532015112830366", "CREDIT_CARD"),
        ("user123@paytm", "IN_UPI"),
        ("SBIN0001234", "IN_IFSC"),
        ("12345678901234", "IN_BANK_ACCOUNT"),
    ]
    
    strategies = [
        ("REDACT", RedactStrategy()),
        ("PARTIAL", PartialMaskStrategy()),
        ("TOKENIZE", TokenizeStrategy()),
        ("FPE", FPEStrategy()),
    ]
    
    for strategy_name, strategy in strategies:
        print(f"\n{strategy_name} Strategy:")
        print("-" * 60)
        for value, pii_type in test_cases:
            masked = strategy.mask(value, pii_type)
            print(f"  {pii_type:20s}: {value:25s} → {masked}")
    
    # Test factory
    print("\n\nFactory Test:")
    print("-" * 60)
    for name in ["REDACT", "PARTIAL", "TOKENIZE", "FPE"]:
        strategy = get_masking_strategy(name)
        print(f"  {name}: {strategy.get_name()}")
