"""
Filesystem Masking Adapter - Mask PII in Files
==============================================
Masks PII in filesystem files (CSV, JSON, TXT, etc.)

Supported Formats:
- CSV: Column-aware masking
- JSON: Path-aware masking
- TXT: Position-based masking
- XML: Tag-aware masking (future)
"""

import os
import shutil
import csv
import json
import time
from typing import List, Dict, Optional
from datetime import datetime
from pathlib import Path

from sdk.masking.adapters.base import (
    BaseMaskingAdapter,
    MaskingFinding,
    MaskingResult,
    MaskingStatus
)


class FilesystemMaskingAdapter(BaseMaskingAdapter):
    """
    Masks PII in filesystem files.
    
    Features:
    - Automatic backup creation
    - Format-specific masking (CSV, JSON, TXT)
    - Position-based replacement
    - Verification after masking
    """
    
    def __init__(self, backup_dir: str = "./backups", backup_enabled: bool = True, dry_run: bool = False):
        """
        Initialize filesystem adapter.
        
        Args:
            backup_dir: Directory to store backups
            backup_enabled: Whether to create backups
            dry_run: If True, simulate masking without modifying files
        """
        super().__init__(backup_enabled, dry_run)
        self.backup_dir = backup_dir
        
        # Create backup directory if it doesn't exist
        if backup_enabled and not dry_run:
            os.makedirs(backup_dir, exist_ok=True)
    
    def get_adapter_name(self) -> str:
        return "filesystem"
    
    def create_backup(self, source_location: str) -> Optional[str]:
        """
        Create backup of a file.
        
        Args:
            source_location: Path to file to backup
            
        Returns:
            Backup file path, or None if backup failed
        """
        if not self.backup_enabled:
            self._log("Backup disabled, skipping", "WARNING")
            return None
        
        if self.dry_run:
            self._log(f"[DRY RUN] Would create backup of {source_location}", "INFO")
            return f"{self.backup_dir}/dry_run_backup"
        
        try:
            # Generate backup filename with timestamp
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            filename = Path(source_location).name
            backup_path = os.path.join(self.backup_dir, f"{filename}.{timestamp}.backup")
            
            # Copy file to backup location
            shutil.copy2(source_location, backup_path)
            
            self._log(f"Created backup: {backup_path}")
            return backup_path
            
        except Exception as e:
            self._log(f"Failed to create backup: {e}", "ERROR")
            return None
    
    def mask_findings(
        self,
        findings: List[MaskingFinding],
        masking_strategy,
        source_location: str,
        **kwargs
    ) -> MaskingResult:
        """
        Mask PII findings in a file.
        
        Args:
            findings: List of findings to mask
            masking_strategy: Strategy to use for masking
            source_location: Path to file to mask
            **kwargs: Additional parameters
            
        Returns:
            MaskingResult with operation details
        """
        start_time = time.time()
        
        # Determine file format
        file_ext = Path(source_location).suffix.lower()
        
        # Create backup
        backup_location = None
        if self.backup_enabled:
            backup_location = self.create_backup(source_location)
            if not backup_location and not self.dry_run:
                return MaskingResult(
                    status=MaskingStatus.FAILED,
                    total_findings=len(findings),
                    masked_count=0,
                    failed_count=len(findings),
                    error_message="Failed to create backup"
                )
        
        # Route to format-specific masking
        try:
            if file_ext == '.csv':
                result = self._mask_csv(findings, masking_strategy, source_location)
            elif file_ext == '.json':
                result = self._mask_json(findings, masking_strategy, source_location)
            elif file_ext in ['.txt', '.log', '.md']:
                result = self._mask_text(findings, masking_strategy, source_location)
            else:
                # Default to text-based masking
                self._log(f"Unknown file type {file_ext}, using text-based masking", "WARNING")
                result = self._mask_text(findings, masking_strategy, source_location)
            
            # Update result with backup location and duration
            result.backup_location = backup_location
            result.duration_seconds = time.time() - start_time
            
            return result
            
        except Exception as e:
            self._log(f"Masking failed: {e}", "ERROR")
            return MaskingResult(
                status=MaskingStatus.FAILED,
                total_findings=len(findings),
                masked_count=0,
                failed_count=len(findings),
                backup_location=backup_location,
                error_message=str(e),
                duration_seconds=time.time() - start_time
            )
    
    def _mask_csv(
        self,
        findings: List[MaskingFinding],
        masking_strategy,
        file_path: str
    ) -> MaskingResult:
        """Mask PII in CSV file"""
        if self.dry_run:
            self._log(f"[DRY RUN] Would mask {len(findings)} findings in CSV: {file_path}")
            return MaskingResult(
                status=MaskingStatus.COMPLETED,
                total_findings=len(findings),
                masked_count=len(findings),
                failed_count=0
            )
        
        # Group findings by location (row, column)
        findings_by_location: Dict[str, List[MaskingFinding]] = {}
        for finding in findings:
            findings_by_location.setdefault(finding.location, []).append(finding)
        
        masked_count = 0
        failed_count = 0
        
        try:
            # Read CSV
            with open(file_path, 'r', encoding='utf-8') as f:
                reader = csv.DictReader(f)
                rows = list(reader)
                fieldnames = reader.fieldnames
            
            # Apply masking
            for location, location_findings in findings_by_location.items():
                # Parse location (e.g., "row_5_column_email")
                parts = location.split('_')
                if len(parts) >= 4:
                    row_idx = int(parts[1])
                    column_name = '_'.join(parts[3:])
                    
                    if row_idx < len(rows) and column_name in rows[row_idx]:
                        for finding in location_findings:
                            try:
                                # Apply masking strategy
                                masked_value = masking_strategy.mask(finding.value, finding.pii_type)
                                rows[row_idx][column_name] = masked_value
                                masked_count += 1
                            except Exception as e:
                                self._log(f"Failed to mask finding: {e}", "ERROR")
                                failed_count += 1
            
            # Write masked CSV
            with open(file_path, 'w', encoding='utf-8', newline='') as f:
                writer = csv.DictWriter(f, fieldnames=fieldnames)
                writer.writeheader()
                writer.writerows(rows)
            
            self._log(f"Masked {masked_count}/{len(findings)} findings in CSV")
            
            return MaskingResult(
                status=MaskingStatus.COMPLETED,
                total_findings=len(findings),
                masked_count=masked_count,
                failed_count=failed_count
            )
            
        except Exception as e:
            self._log(f"CSV masking failed: {e}", "ERROR")
            return MaskingResult(
                status=MaskingStatus.FAILED,
                total_findings=len(findings),
                masked_count=masked_count,
                failed_count=len(findings) - masked_count,
                error_message=str(e)
            )
    
    def _mask_json(
        self,
        findings: List[MaskingFinding],
        masking_strategy,
        file_path: str
    ) -> MaskingResult:
        """Mask PII in JSON file"""
        if self.dry_run:
            self._log(f"[DRY RUN] Would mask {len(findings)} findings in JSON: {file_path}")
            return MaskingResult(
                status=MaskingStatus.COMPLETED,
                total_findings=len(findings),
                masked_count=len(findings),
                failed_count=0
            )
        
        masked_count = 0
        failed_count = 0
        
        try:
            # Read JSON
            with open(file_path, 'r', encoding='utf-8') as f:
                data = json.load(f)
            
            # Apply masking
            for finding in findings:
                try:
                    # Parse JSON path (e.g., "users[0].email")
                    path = finding.location
                    masked_value = masking_strategy.mask(finding.value, finding.pii_type)
                    
                    # Navigate to the value and replace it
                    self._set_json_value(data, path, masked_value)
                    masked_count += 1
                    
                except Exception as e:
                    self._log(f"Failed to mask finding at {finding.location}: {e}", "ERROR")
                    failed_count += 1
            
            # Write masked JSON
            with open(file_path, 'w', encoding='utf-8') as f:
                json.dump(data, f, indent=2, ensure_ascii=False)
            
            self._log(f"Masked {masked_count}/{len(findings)} findings in JSON")
            
            return MaskingResult(
                status=MaskingStatus.COMPLETED,
                total_findings=len(findings),
                masked_count=masked_count,
                failed_count=failed_count
            )
            
        except Exception as e:
            self._log(f"JSON masking failed: {e}", "ERROR")
            return MaskingResult(
                status=MaskingStatus.FAILED,
                total_findings=len(findings),
                masked_count=masked_count,
                failed_count=len(findings) - masked_count,
                error_message=str(e)
            )
    
    def _mask_text(
        self,
        findings: List[MaskingFinding],
        masking_strategy,
        file_path: str
    ) -> MaskingResult:
        """Mask PII in text file using position-based replacement"""
        if self.dry_run:
            self._log(f"[DRY RUN] Would mask {len(findings)} findings in text: {file_path}")
            return MaskingResult(
                status=MaskingStatus.COMPLETED,
                total_findings=len(findings),
                masked_count=len(findings),
                failed_count=0
            )
        
        masked_count = 0
        failed_count = 0
        
        try:
            # Read file content
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read()
            
            # Sort findings by position (descending) to avoid offset issues
            sorted_findings = sorted(
                [f for f in findings if f.start_pos is not None and f.end_pos is not None],
                key=lambda x: x.start_pos,
                reverse=True
            )
            
            # Apply masking from end to start
            for finding in sorted_findings:
                try:
                    masked_value = masking_strategy.mask(finding.value, finding.pii_type)
                    content = content[:finding.start_pos] + masked_value + content[finding.end_pos:]
                    masked_count += 1
                except Exception as e:
                    self._log(f"Failed to mask finding: {e}", "ERROR")
                    failed_count += 1
            
            # Write masked content
            with open(file_path, 'w', encoding='utf-8') as f:
                f.write(content)
            
            self._log(f"Masked {masked_count}/{len(findings)} findings in text file")
            
            return MaskingResult(
                status=MaskingStatus.COMPLETED,
                total_findings=len(findings),
                masked_count=masked_count,
                failed_count=failed_count
            )
            
        except Exception as e:
            self._log(f"Text masking failed: {e}", "ERROR")
            return MaskingResult(
                status=MaskingStatus.FAILED,
                total_findings=len(findings),
                masked_count=masked_count,
                failed_count=len(findings) - masked_count,
                error_message=str(e)
            )
    
    def _set_json_value(self, data, path: str, value):
        """Set value in JSON data using path notation"""
        # Simple implementation for basic paths
        # TODO: Implement full JSONPath support for complex paths
        keys = path.replace('[', '.').replace(']', '').split('.')
        current = data
        
        for key in keys[:-1]:
            if key.isdigit():
                current = current[int(key)]
            else:
                current = current[key]
        
        final_key = keys[-1]
        if final_key.isdigit():
            current[int(final_key)] = value
        else:
            current[final_key] = value
    
    def rollback(self, backup_location: str, target_location: str) -> bool:
        """
        Rollback masking by restoring from backup.
        
        Args:
            backup_location: Path to backup file
            target_location: Path to restore to
            
        Returns:
            True if rollback successful
        """
        if self.dry_run:
            self._log(f"[DRY RUN] Would rollback from {backup_location} to {target_location}")
            return True
        
        try:
            shutil.copy2(backup_location, target_location)
            self._log(f"Rolled back from backup: {backup_location}")
            return True
        except Exception as e:
            self._log(f"Rollback failed: {e}", "ERROR")
            return False
    
    def verify_masking(self, source_location: str, findings: List[MaskingFinding]) -> bool:
        """
        Verify that PII values are no longer present in the file.
        
        Args:
            source_location: Path to masked file
            findings: Original findings that should be masked
            
        Returns:
            True if all PII values are masked
        """
        try:
            with open(source_location, 'r', encoding='utf-8') as f:
                content = f.read()
            
            # Check if any original PII values still exist
            for finding in findings:
                if finding.value in content:
                    self._log(f"Verification failed: Found unmasked value: {finding.value[:10]}***", "ERROR")
                    return False
            
            self._log("Verification passed: All PII values masked")
            return True
            
        except Exception as e:
            self._log(f"Verification failed: {e}", "ERROR")
            return False


if __name__ == "__main__":
    print("=== Filesystem Masking Adapter Test ===\n")
    
    from sdk.masking.strategies import PartialMaskStrategy
    
    # Create test CSV file
    test_csv = "/tmp/test_data.csv"
    with open(test_csv, 'w') as f:
        f.write("name,email,phone\n")
        f.write("John Doe,john@example.com,9876543210\n")
        f.write("Jane Smith,jane@company.com,8765432109\n")
    
    print(f"Created test file: {test_csv}")
    
    # Create adapter
    adapter = FilesystemMaskingAdapter(backup_dir="/tmp/backups")
    
    # Create test findings
    findings = [
        MaskingFinding(
            value="john@example.com",
            pii_type="EMAIL_ADDRESS",
            location="row_0_column_email"
        ),
        MaskingFinding(
            value="9876543210",
            pii_type="IN_PHONE",
            location="row_0_column_phone"
        ),
    ]
    
    # Apply masking
    strategy = PartialMaskStrategy()
    result = adapter.mask_findings(findings, strategy, test_csv)
    
    print(f"\nMasking Result:")
    print(f"  Status: {result.status.value}")
    print(f"  Masked: {result.masked_count}/{result.total_findings}")
    print(f"  Backup: {result.backup_location}")
    print(f"  Duration: {result.duration_seconds:.2f}s")
    
    # Verify
    verified = adapter.verify_masking(test_csv, findings)
    print(f"\nVerification: {'✓ PASSED' if verified else '✗ FAILED'}")
    
    # Show masked content
    print(f"\nMasked CSV content:")
    with open(test_csv, 'r') as f:
        print(f.read())
