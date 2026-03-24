"""
Base Masking Adapter - Interface for source-specific masking
============================================================
Defines the interface that all masking adapters must implement.

Each adapter is responsible for masking PII in a specific data source type
(filesystem, PostgreSQL, MySQL, MongoDB, S3, GCS, Redis, etc.)
"""

from abc import ABC, abstractmethod
from typing import List, Dict, Optional
from dataclasses import dataclass, field
from datetime import datetime
from enum import Enum


class MaskingStatus(Enum):
    """Status of a masking operation"""
    PENDING = "pending"
    IN_PROGRESS = "in_progress"
    COMPLETED = "completed"
    FAILED = "failed"
    ROLLED_BACK = "rolled_back"


@dataclass
class MaskingFinding:
    """Represents a PII finding to be masked"""
    value: str
    pii_type: str
    location: str  # File path, table.column, etc.
    start_pos: Optional[int] = None  # For text-based sources
    end_pos: Optional[int] = None
    metadata: Dict = field(default_factory=dict)


@dataclass
class MaskingResult:
    """Result of a masking operation"""
    status: MaskingStatus
    total_findings: int
    masked_count: int
    failed_count: int
    backup_location: Optional[str] = None
    error_message: Optional[str] = None
    duration_seconds: float = 0.0
    timestamp: datetime = field(default_factory=datetime.now)
    details: Dict = field(default_factory=dict)
    
    def to_dict(self) -> dict:
        """Convert to dictionary"""
        return {
            "status": self.status.value,
            "total_findings": self.total_findings,
            "masked_count": self.masked_count,
            "failed_count": self.failed_count,
            "success_rate": f"{(self.masked_count / self.total_findings * 100):.1f}%" if self.total_findings > 0 else "N/A",
            "backup_location": self.backup_location,
            "error_message": self.error_message,
            "duration_seconds": self.duration_seconds,
            "timestamp": self.timestamp.isoformat(),
            "details": self.details,
        }


class BaseMaskingAdapter(ABC):
    """
    Base class for all masking adapters.
    
    Each adapter implements source-specific masking logic while following
    a common interface for backup, masking, and rollback operations.
    """
    
    def __init__(self, backup_enabled: bool = True, dry_run: bool = False):
        """
        Initialize adapter.
        
        Args:
            backup_enabled: Whether to create backups before masking
            dry_run: If True, simulate masking without modifying data
        """
        self.backup_enabled = backup_enabled
        self.dry_run = dry_run
    
    @abstractmethod
    def get_adapter_name(self) -> str:
        """Get adapter name (e.g., 'filesystem', 'postgresql')"""
        pass
    
    @abstractmethod
    def create_backup(self, source_location: str) -> Optional[str]:
        """
        Create backup of source data before masking.
        
        Args:
            source_location: Location of data to backup (file path, table name, etc.)
            
        Returns:
            Backup location/identifier, or None if backup failed
        """
        pass
    
    @abstractmethod
    def mask_findings(
        self,
        findings: List[MaskingFinding],
        masking_strategy,  # MaskingStrategy instance
        source_location: str,
        **kwargs
    ) -> MaskingResult:
        """
        Mask PII findings in the source data.
        
        Args:
            findings: List of findings to mask
            masking_strategy: Strategy to use for masking
            source_location: Location of data to mask
            **kwargs: Adapter-specific parameters
            
        Returns:
            MaskingResult with operation details
        """
        pass
    
    @abstractmethod
    def rollback(self, backup_location: str, target_location: str) -> bool:
        """
        Rollback masking by restoring from backup.
        
        Args:
            backup_location: Location of backup
            target_location: Location to restore to
            
        Returns:
            True if rollback successful, False otherwise
        """
        pass
    
    @abstractmethod
    def verify_masking(self, source_location: str, findings: List[MaskingFinding]) -> bool:
        """
        Verify that masking was successful.
        
        Args:
            source_location: Location of masked data
            findings: Original findings that should be masked
            
        Returns:
            True if all findings are masked, False otherwise
        """
        pass
    
    def _log(self, message: str, level: str = "INFO"):
        """Log message"""
        timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        prefix = "🔒" if level == "INFO" else "⚠️" if level == "WARNING" else "❌"
        print(f"[{timestamp}] {prefix} [{self.get_adapter_name()}] {message}")


if __name__ == "__main__":
    print("=== Base Masking Adapter ===\n")
    print("This is an abstract base class. See specific adapters for implementations.")
    print("\nAvailable adapter types:")
    print("  - FilesystemMaskingAdapter (CSV, JSON, TXT files)")
    print("  - PostgreSQLMaskingAdapter (PostgreSQL databases)")
    print("  - MySQLMaskingAdapter (MySQL databases)")
    print("  - MongoDBMaskingAdapter (MongoDB collections)")
    print("  - S3MaskingAdapter (AWS S3 buckets)")
    print("  - GCSMaskingAdapter (Google Cloud Storage)")
    print("  - RedisMaskingAdapter (Redis key-value store)")
