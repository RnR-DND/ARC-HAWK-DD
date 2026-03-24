"""
PostgreSQL Masking Adapter - Mask PII in PostgreSQL Databases
=============================================================
Masks PII in PostgreSQL tables using UPDATE queries.

Features:
- Transaction-based masking (rollback on error)
- Backup table creation
- Row-level masking with WHERE clauses
- Connection pooling support
"""

import time
from typing import List, Dict, Optional
from datetime import datetime

try:
    import psycopg2
    from psycopg2 import sql
    from psycopg2.extras import RealDictCursor
    PSYCOPG2_AVAILABLE = True
except ImportError:
    PSYCOPG2_AVAILABLE = False

from sdk.masking.adapters.base import (
    BaseMaskingAdapter,
    MaskingFinding,
    MaskingResult,
    MaskingStatus
)


class PostgreSQLMaskingAdapter(BaseMaskingAdapter):
    """
    Masks PII in PostgreSQL databases.
    
    Features:
    - Automatic backup table creation
    - Transaction-based operations
    - Row-level UPDATE queries
    - Rollback support
    """
    
    def __init__(
        self,
        connection_params: Dict[str, str],
        backup_enabled: bool = True,
        dry_run: bool = False
    ):
        """
        Initialize PostgreSQL adapter.
        
        Args:
            connection_params: Database connection parameters
                {
                    'host': 'localhost',
                    'port': '5432',
                    'database': 'mydb',
                    'user': 'user',
                    'password': 'pass'
                }
            backup_enabled: Whether to create backup tables
            dry_run: If True, simulate masking without modifying data
        """
        if not PSYCOPG2_AVAILABLE:
            raise ImportError("psycopg2 is required for PostgreSQL masking. Install with: pip install psycopg2-binary")
        
        super().__init__(backup_enabled, dry_run)
        self.connection_params = connection_params
        self.connection = None
    
    def get_adapter_name(self) -> str:
        return "postgresql"
    
    def _connect(self):
        """Establish database connection"""
        if not self.connection or self.connection.closed:
            self.connection = psycopg2.connect(**self.connection_params)
        return self.connection
    
    def _close(self):
        """Close database connection"""
        if self.connection and not self.connection.closed:
            self.connection.close()
    
    def create_backup(self, source_location: str) -> Optional[str]:
        """
        Create backup table.
        
        Args:
            source_location: Table name (e.g., "public.users")
            
        Returns:
            Backup table name, or None if backup failed
        """
        if not self.backup_enabled:
            self._log("Backup disabled, skipping", "WARNING")
            return None
        
        if self.dry_run:
            backup_table = f"{source_location}_backup_{datetime.now().strftime('%Y%m%d_%H%M%S')}"
            self._log(f"[DRY RUN] Would create backup table: {backup_table}")
            return backup_table
        
        try:
            conn = self._connect()
            cursor = conn.cursor()
            
            # Generate backup table name
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            backup_table = f"{source_location}_backup_{timestamp}"
            
            # Create backup table as copy of original
            query = sql.SQL("CREATE TABLE {backup} AS SELECT * FROM {original}").format(
                backup=sql.Identifier(backup_table),
                original=sql.Identifier(source_location)
            )
            
            cursor.execute(query)
            conn.commit()
            
            self._log(f"Created backup table: {backup_table}")
            return backup_table
            
        except Exception as e:
            self._log(f"Failed to create backup: {e}", "ERROR")
            if conn:
                conn.rollback()
            return None
        finally:
            if cursor:
                cursor.close()
    
    def mask_findings(
        self,
        findings: List[MaskingFinding],
        masking_strategy,
        source_location: str,
        **kwargs
    ) -> MaskingResult:
        """
        Mask PII findings in a PostgreSQL table.
        
        Args:
            findings: List of findings to mask
            masking_strategy: Strategy to use for masking
            source_location: Table name (e.g., "public.users")
            **kwargs: Additional parameters
                - primary_key: Primary key column name (default: "id")
            
        Returns:
            MaskingResult with operation details
        """
        start_time = time.time()
        primary_key = kwargs.get('primary_key', 'id')
        
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
        
        if self.dry_run:
            self._log(f"[DRY RUN] Would mask {len(findings)} findings in table: {source_location}")
            return MaskingResult(
                status=MaskingStatus.COMPLETED,
                total_findings=len(findings),
                masked_count=len(findings),
                failed_count=0,
                backup_location=backup_location,
                duration_seconds=time.time() - start_time
            )
        
        # Group findings by table.column
        findings_by_column: Dict[str, List[MaskingFinding]] = {}
        for finding in findings:
            # Location format: "table.column" or "schema.table.column"
            findings_by_column.setdefault(finding.location, []).append(finding)
        
        masked_count = 0
        failed_count = 0
        
        try:
            conn = self._connect()
            cursor = conn.cursor()
            
            # Process each column
            for location, column_findings in findings_by_column.items():
                # Parse location
                parts = location.split('.')
                column_name = parts[-1]
                table_name = parts[-2] if len(parts) >= 2 else source_location
                
                # Apply masking for each finding
                for finding in column_findings:
                    try:
                        masked_value = masking_strategy.mask(finding.value, finding.pii_type)
                        
                        # Build UPDATE query with WHERE clause
                        # UPDATE table SET column = masked_value WHERE column = original_value
                        query = sql.SQL(
                            "UPDATE {table} SET {column} = %s WHERE {column} = %s"
                        ).format(
                            table=sql.Identifier(table_name),
                            column=sql.Identifier(column_name)
                        )
                        
                        cursor.execute(query, (masked_value, finding.value))
                        
                        if cursor.rowcount > 0:
                            masked_count += cursor.rowcount
                            self._log(f"Masked {cursor.rowcount} rows in {table_name}.{column_name}")
                        else:
                            self._log(f"No rows matched for value in {table_name}.{column_name}", "WARNING")
                            failed_count += 1
                        
                    except Exception as e:
                        self._log(f"Failed to mask finding in {location}: {e}", "ERROR")
                        failed_count += 1
            
            # Commit transaction
            conn.commit()
            
            self._log(f"Masked {masked_count} rows across {len(findings)} findings")
            
            return MaskingResult(
                status=MaskingStatus.COMPLETED,
                total_findings=len(findings),
                masked_count=masked_count,
                failed_count=failed_count,
                backup_location=backup_location,
                duration_seconds=time.time() - start_time
            )
            
        except Exception as e:
            self._log(f"Masking failed: {e}", "ERROR")
            if conn:
                conn.rollback()
            
            return MaskingResult(
                status=MaskingStatus.FAILED,
                total_findings=len(findings),
                masked_count=masked_count,
                failed_count=len(findings) - masked_count,
                backup_location=backup_location,
                error_message=str(e),
                duration_seconds=time.time() - start_time
            )
        finally:
            if cursor:
                cursor.close()
    
    def rollback(self, backup_location: str, target_location: str) -> bool:
        """
        Rollback masking by restoring from backup table.
        
        Args:
            backup_location: Backup table name
            target_location: Original table name
            
        Returns:
            True if rollback successful
        """
        if self.dry_run:
            self._log(f"[DRY RUN] Would rollback from {backup_location} to {target_location}")
            return True
        
        try:
            conn = self._connect()
            cursor = conn.cursor()
            
            # Delete current data
            delete_query = sql.SQL("DELETE FROM {table}").format(
                table=sql.Identifier(target_location)
            )
            cursor.execute(delete_query)
            
            # Restore from backup
            restore_query = sql.SQL(
                "INSERT INTO {target} SELECT * FROM {backup}"
            ).format(
                target=sql.Identifier(target_location),
                backup=sql.Identifier(backup_location)
            )
            cursor.execute(restore_query)
            
            conn.commit()
            
            self._log(f"Rolled back from backup: {backup_location}")
            return True
            
        except Exception as e:
            self._log(f"Rollback failed: {e}", "ERROR")
            if conn:
                conn.rollback()
            return False
        finally:
            if cursor:
                cursor.close()
    
    def verify_masking(self, source_location: str, findings: List[MaskingFinding]) -> bool:
        """
        Verify that PII values are no longer present in the table.
        
        Args:
            source_location: Table name
            findings: Original findings that should be masked
            
        Returns:
            True if all PII values are masked
        """
        try:
            conn = self._connect()
            cursor = conn.cursor()
            
            # Check if any original PII values still exist
            for finding in findings:
                # Parse location
                parts = finding.location.split('.')
                column_name = parts[-1]
                table_name = parts[-2] if len(parts) >= 2 else source_location
                
                # Query for original value
                query = sql.SQL(
                    "SELECT COUNT(*) FROM {table} WHERE {column} = %s"
                ).format(
                    table=sql.Identifier(table_name),
                    column=sql.Identifier(column_name)
                )
                
                cursor.execute(query, (finding.value,))
                count = cursor.fetchone()[0]
                
                if count > 0:
                    self._log(f"Verification failed: Found {count} unmasked rows in {table_name}.{column_name}", "ERROR")
                    return False
            
            self._log("Verification passed: All PII values masked")
            return True
            
        except Exception as e:
            self._log(f"Verification failed: {e}", "ERROR")
            return False
        finally:
            if cursor:
                cursor.close()
    
    def __del__(self):
        """Cleanup: close connection"""
        self._close()


if __name__ == "__main__":
    print("=== PostgreSQL Masking Adapter ===\n")
    print("This adapter requires a PostgreSQL database connection.")
    print("\nUsage example:")
    print("""
from sdk.masking.adapters.postgresql import PostgreSQLMaskingAdapter
from sdk.masking.strategies import PartialMaskStrategy

# Connection parameters
conn_params = {
    'host': 'localhost',
    'port': '5432',
    'database': 'mydb',
    'user': 'user',
    'password': 'password'
}

# Create adapter
adapter = PostgreSQLMaskingAdapter(conn_params)

# Create findings
findings = [
    MaskingFinding(
        value="john@example.com",
        pii_type="EMAIL_ADDRESS",
        location="public.users.email"
    ),
]

# Apply masking
strategy = PartialMaskStrategy()
result = adapter.mask_findings(findings, strategy, "public.users")

print(f"Masked: {result.masked_count}/{result.total_findings}")
    """)
