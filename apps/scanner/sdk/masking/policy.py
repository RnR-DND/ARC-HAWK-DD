"""
Masking Policy Engine - Define and Enforce Masking Policies
===========================================================
Manages masking policies that define which PII types to mask and how.

Features:
- Per-PII-type strategy configuration
- Asset exclusion rules
- Backup and dry-run settings
- Policy validation
"""

from typing import Dict, List, Optional, Set
from dataclasses import dataclass, field
from enum import Enum
import yaml
import json

from sdk.masking.strategies import MaskingStrategy, get_masking_strategy


class PolicyMode(Enum):
    """Policy enforcement mode"""
    STRICT = "strict"  # Fail if any finding cannot be masked
    LENIENT = "lenient"  # Continue even if some findings fail
    DRY_RUN = "dry_run"  # Simulate without making changes


@dataclass
class MaskingPolicy:
    """
    Defines masking policy for PII data.
    
    A policy specifies:
    - Which PII types to mask
    - Which masking strategy to use for each type
    - Which assets to exclude
    - Backup and safety settings
    """
    
    # Policy metadata
    name: str
    description: str = ""
    mode: PolicyMode = PolicyMode.STRICT
    
    # PII type strategies
    pii_type_strategies: Dict[str, str] = field(default_factory=dict)
    
    # Default strategy for types not explicitly configured
    default_strategy: str = "PARTIAL"
    
    # Backup settings
    backup_enabled: bool = True
    backup_retention_days: int = 30
    
    # Exclusion rules
    excluded_assets: Set[str] = field(default_factory=set)
    excluded_pii_types: Set[str] = field(default_factory=set)
    
    # Safety settings
    require_confirmation: bool = True
    max_findings_per_run: int = 10000
    
    # Secret key for tokenization/FPE
    secret_key: Optional[str] = None
    
    def should_mask_pii_type(self, pii_type: str) -> bool:
        """Check if a PII type should be masked"""
        return pii_type not in self.excluded_pii_types
    
    def should_mask_asset(self, asset_path: str) -> bool:
        """Check if an asset should be masked"""
        # Check exact match
        if asset_path in self.excluded_assets:
            return False
        
        # Check pattern match (simple glob-like)
        for pattern in self.excluded_assets:
            if '*' in pattern:
                # Simple wildcard matching
                pattern_parts = pattern.split('*')
                if all(part in asset_path for part in pattern_parts if part):
                    return False
        
        return True
    
    def get_strategy_for_pii_type(self, pii_type: str) -> str:
        """Get masking strategy for a PII type"""
        return self.pii_type_strategies.get(pii_type, self.default_strategy)
    
    def get_masking_strategy_instance(self, pii_type: str) -> MaskingStrategy:
        """Get masking strategy instance for a PII type"""
        strategy_name = self.get_strategy_for_pii_type(pii_type)
        return get_masking_strategy(strategy_name, self.secret_key)
    
    def validate(self) -> List[str]:
        """
        Validate policy configuration.
        
        Returns:
            List of validation errors (empty if valid)
        """
        errors = []
        
        # Validate strategy names
        valid_strategies = {"REDACT", "PARTIAL", "TOKENIZE", "FPE"}
        for pii_type, strategy in self.pii_type_strategies.items():
            if strategy.upper() not in valid_strategies:
                errors.append(f"Invalid strategy '{strategy}' for PII type '{pii_type}'")
        
        if self.default_strategy.upper() not in valid_strategies:
            errors.append(f"Invalid default strategy '{self.default_strategy}'")
        
        # Validate retention days
        if self.backup_retention_days < 0:
            errors.append("Backup retention days must be >= 0")
        
        # Validate max findings
        if self.max_findings_per_run <= 0:
            errors.append("Max findings per run must be > 0")
        
        return errors
    
    def to_dict(self) -> dict:
        """Convert policy to dictionary"""
        return {
            "name": self.name,
            "description": self.description,
            "mode": self.mode.value,
            "pii_type_strategies": self.pii_type_strategies,
            "default_strategy": self.default_strategy,
            "backup_enabled": self.backup_enabled,
            "backup_retention_days": self.backup_retention_days,
            "excluded_assets": list(self.excluded_assets),
            "excluded_pii_types": list(self.excluded_pii_types),
            "require_confirmation": self.require_confirmation,
            "max_findings_per_run": self.max_findings_per_run,
        }
    
    @classmethod
    def from_dict(cls, data: dict) -> 'MaskingPolicy':
        """Create policy from dictionary"""
        return cls(
            name=data["name"],
            description=data.get("description", ""),
            mode=PolicyMode(data.get("mode", "strict")),
            pii_type_strategies=data.get("pii_type_strategies", {}),
            default_strategy=data.get("default_strategy", "PARTIAL"),
            backup_enabled=data.get("backup_enabled", True),
            backup_retention_days=data.get("backup_retention_days", 30),
            excluded_assets=set(data.get("excluded_assets", [])),
            excluded_pii_types=set(data.get("excluded_pii_types", [])),
            require_confirmation=data.get("require_confirmation", True),
            max_findings_per_run=data.get("max_findings_per_run", 10000),
            secret_key=data.get("secret_key"),
        )
    
    @classmethod
    def from_yaml(cls, filepath: str) -> 'MaskingPolicy':
        """Load policy from YAML file"""
        with open(filepath, 'r') as f:
            data = yaml.safe_load(f)
        return cls.from_dict(data)
    
    @classmethod
    def from_json(cls, filepath: str) -> 'MaskingPolicy':
        """Load policy from JSON file"""
        with open(filepath, 'r') as f:
            data = json.load(f)
        return cls.from_dict(data)
    
    def save_yaml(self, filepath: str):
        """Save policy to YAML file"""
        with open(filepath, 'w') as f:
            yaml.dump(self.to_dict(), f, default_flow_style=False)
    
    def save_json(self, filepath: str):
        """Save policy to JSON file"""
        with open(filepath, 'w') as f:
            json.dump(self.to_dict(), f, indent=2)

    @classmethod
    def default(cls) -> 'MaskingPolicy':
        """
        FIX C8: Return the default masking policy (PARTIAL masking for all PII).
        Convenience classmethod so callers can write MaskingPolicy.default()
        without importing get_default_policy separately.
        """
        return cls(
            name="default",
            description="Default masking policy with partial masking for all PII types",
            mode=PolicyMode.STRICT,
            default_strategy="PARTIAL",
            backup_enabled=True,
            require_confirmation=False,  # Non-interactive; used in scan pipeline
        )


# Predefined policies
def get_default_policy() -> MaskingPolicy:
    """Get default masking policy"""
    return MaskingPolicy(
        name="default",
        description="Default masking policy with partial masking for all PII types",
        mode=PolicyMode.STRICT,
        default_strategy="PARTIAL",
        backup_enabled=True,
        require_confirmation=True,
    )


def get_strict_policy() -> MaskingPolicy:
    """Get strict masking policy (full redaction)"""
    return MaskingPolicy(
        name="strict",
        description="Strict policy with full redaction for all PII types",
        mode=PolicyMode.STRICT,
        pii_type_strategies={
            "IN_AADHAAR": "REDACT",
            "IN_PAN": "REDACT",
            "IN_PASSPORT": "REDACT",
            "CREDIT_CARD": "REDACT",
            "IN_BANK_ACCOUNT": "REDACT",
        },
        default_strategy="REDACT",
        backup_enabled=True,
        require_confirmation=True,
    )


def get_development_policy() -> MaskingPolicy:
    """Get development policy (tokenization for referential integrity)"""
    return MaskingPolicy(
        name="development",
        description="Development policy with tokenization for testing",
        mode=PolicyMode.LENIENT,
        default_strategy="TOKENIZE",
        backup_enabled=True,
        require_confirmation=False,
        excluded_assets={"/data/test/*", "test_*"},
    )


if __name__ == "__main__":
    print("=== Masking Policy Engine Test ===\n")
    
    # Test 1: Create default policy
    print("Test 1: Default Policy")
    policy = get_default_policy()
    print(f"  Name: {policy.name}")
    print(f"  Mode: {policy.mode.value}")
    print(f"  Default Strategy: {policy.default_strategy}")
    
    # Test 2: Validate policy
    print("\nTest 2: Policy Validation")
    errors = policy.validate()
    if errors:
        print(f"  Validation errors: {errors}")
    else:
        print("  ✓ Policy is valid")
    
    # Test 3: Check PII type masking
    print("\nTest 3: PII Type Masking")
    pii_types = ["IN_AADHAAR", "EMAIL_ADDRESS", "IN_PHONE"]
    for pii_type in pii_types:
        should_mask = policy.should_mask_pii_type(pii_type)
        strategy = policy.get_strategy_for_pii_type(pii_type)
        print(f"  {pii_type}: mask={should_mask}, strategy={strategy}")
    
    # Test 4: Check asset exclusion
    print("\nTest 4: Asset Exclusion")
    dev_policy = get_development_policy()
    test_assets = ["/data/test/file.csv", "/data/prod/file.csv", "test_table"]
    for asset in test_assets:
        should_mask = dev_policy.should_mask_asset(asset)
        print(f"  {asset}: mask={should_mask}")
    
    # Test 5: Save and load policy
    print("\nTest 5: Save/Load Policy")
    policy.save_json("/tmp/test_policy.json")
    loaded_policy = MaskingPolicy.from_json("/tmp/test_policy.json")
    print(f"  Saved and loaded policy: {loaded_policy.name}")
    print(f"  ✓ Policy serialization working")
    
    # Test 6: Get masking strategy instance
    print("\nTest 6: Get Strategy Instance")
    strategy = policy.get_masking_strategy_instance("IN_AADHAAR")
    test_value = "234567890124"
    masked = strategy.mask(test_value, "IN_AADHAAR")
    print(f"  Original: {test_value}")
    print(f"  Masked: {masked}")
    print(f"  Strategy: {strategy.get_name()}")
