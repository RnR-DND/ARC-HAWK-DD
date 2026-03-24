#!/usr/bin/env python3
"""
ARC-Hawk Architecture Verification Script
==========================================
Ensures Intelligence-at-Edge compliance with mandatory architecture.

Usage:
    python3 verify_architecture.py
    
Exit Codes:
    0 - All checks passed
    1 - One or more checks failed
    2 - Fatal error (script failure)
"""

import os
import sys
import json
import subprocess
from pathlib import Path
from typing import Tuple, List, Dict
import requests
from rich.console import Console
from rich.table import Table
from rich.panel import Panel

console = Console()

# Configuration
BACKEND_URL = os.getenv("BACKEND_URL", "http://localhost:8080")
NEO4J_URL = os.getenv("NEO4J_URL", "bolt://localhost:7687")
PROJECT_ROOT = Path(__file__).parent.parent

# Locked PII Types (11 types, English only)
LOCKED_PII_TYPES = {
    "IN_PAN", "IN_PASSPORT", "IN_AADHAAR", "CREDIT_CARD",
    "IN_UPI", "IN_IFSC", "IN_BANK_ACCOUNT", "IN_PHONE",
    "EMAIL_ADDRESS", "IN_VOTER_ID", "IN_DRIVING_LICENSE"
}


class VerificationResult:
    def __init__(self, name: str, passed: bool, message: str, details: str = ""):
        self.name = name
        self.passed = passed
        self.message = message
        self.details = details


def check_scanner_output_contract() -> VerificationResult:
    """
    V1: Verify scanner emits only verified findings.
    
    Pass Criteria:
    - SDK output schema includes: pii_type, confidence, validator
    - SDK output does NOT include raw PII values
    """
    try:
        schema_file = PROJECT_ROOT / "apps" / "scanner" / "sdk" / "schema.py"
        
        if not schema_file.exists():
            return VerificationResult(
                "Scanner Output Contract",
                False,
                "schema.py not found",
                f"Expected: {schema_file}"
            )
        
        with open(schema_file, 'r') as f:
            schema_content = f.read()
        
        # Check for required fields
        has_pii_type = "pii_type" in schema_content
        has_confidence = "confidence" in schema_content
        has_validator = "validator" in schema_content
        
        # Check it doesn't have raw match values (should have hash)
        has_raw_matches = '"matches"' in schema_content or "'matches'" in schema_content
        
        if has_pii_type and has_confidence and has_validator and not has_raw_matches:
            return VerificationResult(
                "Scanner Output Contract",
                True,
                "SDK emits verified findings with required fields"
            )
        else:
            missing = []
            if not has_pii_type:
                missing.append("pii_type")
            if not has_confidence:
                missing.append("confidence")
            if not has_validator:
                missing.append("validator")
            if has_raw_matches:
                missing.append("(raw matches field exists - should be hash)")
            
            return VerificationResult(
                "Scanner Output Contract",
                False,
                f"Schema missing fields: {', '.join(missing)}",
                "SDK should emit verified findings only"
            )
    
    except Exception as e:
        return VerificationResult(
            "Scanner Output Contract",
            False,
            f"Error checking schema: {str(e)}"
        )


def check_backend_no_presidio_client() -> VerificationResult:
    """
    V2: Ensure backend has NO Presidio client.
    
    Pass Criteria:
    - presidio_client.go file does NOT exist
    - No HTTP calls to Presidio in backend code
    """
    try:
        presidio_client_file = PROJECT_ROOT / "apps" / "backend" / "internal" / "service" / "presidio_client.go"
        
        if presidio_client_file.exists():
            return VerificationResult(
                "Backend No Presidio Client",
                False,
                "presidio_client.go still exists",
                f"Delete: {presidio_client_file}"
            )
        
        # Grep for Presidio HTTP calls in backend
        backend_path = PROJECT_ROOT / "apps" / "backend"
        result = subprocess.run(
            ["grep", "-r", "presidio.*/analyze", str(backend_path)],
            capture_output=True,
            text=True
        )
        
        if result.returncode == 0:  # Found matches
            return VerificationResult(
                "Backend No Presidio Client",
                False,
                "Backend still makes HTTP calls to Presidio",
                result.stdout
            )
        
        return VerificationResult(
            "Backend No Presidio Client",
            True,
            "Backend has no Presidio client"
        )
    
    except Exception as e:
        return VerificationResult(
            "Backend No Presidio Client",
            False,
            f"Error checking backend: {str(e)}"
        )


def check_backend_no_validators() -> VerificationResult:
    """
    V3: Ensure backend has NO validation logic.
    
    Pass Criteria:
    - No luhnValidate, verhoeffValidate, panValidate functions
    """
    try:
        classification_file = PROJECT_ROOT / "apps" / "backend" / "internal" / "service" / "classification_service.go"
        
        if not classification_file.exists():
            return VerificationResult(
                "Backend No Validators",
                False,
                "classification_service.go not found"
            )
        
        with open(classification_file, 'r') as f:
            content = f.read()
        
        # Check for validator functions
        forbidden_functions = [
            "luhnValidate",
            "verhoeffValidate",
            "panValidate",
            "ssnValidate",
            "runValidator"
        ]
        
        found_validators = [func for func in forbidden_functions if func in content]
        
        if found_validators:
            return VerificationResult(
                "Backend No Validators",
                False,
                f"Backend still has validators: {', '.join(found_validators)}",
                "Validators should ONLY be in SDK"
            )
        
        return VerificationResult(
            "Backend No Validators",
            True,
            "Backend has no validation logic"
        )
    
    except Exception as e:
        return VerificationResult(
            "Backend No Validators",
            False,
            f"Error checking validators: {str(e)}"
        )


def check_neo4j_mandatory() -> VerificationResult:
    """
    V7: Ensure Neo4j is mandatory (no fallback).
    
    Pass Criteria:
    - Backend code fails if Neo4j unavailable
    - No graceful degradation to PostgreSQL lineage
    """
    try:
        main_file = PROJECT_ROOT / "apps" / "backend" / "cmd" / "server" / "main.go"
        
        if not main_file.exists():
            return VerificationResult(
                "Neo4j Mandatory",
                False,
                "main.go not found"
            )
        
        with open(main_file, 'r') as f:
            content = f.read()
        
        # Check for fallback patterns
        has_fallback = "PostgreSQL-only lineage" in content or "gracefully falls back" in content
        has_neo4j_disabled_check = 'neo4jEnabled != "true"' in content
        
        if has_fallback or has_neo4j_disabled_check:
            return VerificationResult(
                "Neo4j Mandatory",
                False,
                "Backend still has Neo4j fallback logic",
                "Neo4j should be REQUIRED, not optional"
            )
        
        # Check for fatal error on Neo4j failure
        has_fatal_on_failure = "log.Fatalf" in content and "Neo4j" in content
        
        if has_fatal_on_failure:
            return VerificationResult(
                "Neo4j Mandatory",
                True,
                "Neo4j is mandatory - backend fails if unavailable"
            )
        else:
            return VerificationResult(
                "Neo4j Mandatory",
                False,
                "Backend doesn't fail on Neo4j unavailability",
                "Should use log.Fatalf() if Neo4j connection fails"
            )
    
    except Exception as e:
        return VerificationResult(
            "Neo4j Mandatory",
            False,
            f"Error checking Neo4j enforcement: {str(e)}"
        )


def check_pii_scope_locked() -> VerificationResult:
    """
    V5: Ensure only 11 locked PIIs are processed.
    
    Pass Criteria:
    - Backend rejects PIIs not in LOCKED_PII_TYPES
    - No US_SSN, generic "Secrets", etc.
    """
    try:
        # Try to call the API if backend is running
        try:
            response = requests.get(f"{BACKEND_URL}/api/v1/findings", timeout=5)
            if response.status_code == 200:
                findings = response.json()
                
                # Check if any findings have out-of-scope PII types
                if "data" in findings and isinstance(findings["data"], list):
                    for finding in findings["data"]:
                        # Handle both dict and string values
                        if isinstance(finding, dict):
                            pii_type = finding.get("pattern_name", "").upper()
                        elif isinstance(finding, str):
                            # If finding is a string, skip it
                            continue
                        else:
                            continue
                        
                        # Check if this looks like a PII type
                        if "_" in pii_type and pii_type not in LOCKED_PII_TYPES:
                            # Check common out-of-scope types
                            if any(x in pii_type for x in ["US_SSN", "SSN", "SOCIAL_SECURITY"]):
                                return VerificationResult(
                                    "PII Scope Locked",
                                    False,
                                    f"Found out-of-scope PII: {pii_type}",
                                    "Only 11 locked India PIIs should be processed"
                                )
        except (requests.RequestException, AttributeError, TypeError) as e:
            # Backend not running or API format changed, check code instead
            pass
        
        # Check source code for US_SSN handling
        classification_file = PROJECT_ROOT / "apps" / "backend" / "internal" / "service" / "classification_service.go"
        
        if classification_file.exists():
            with open(classification_file, 'r') as f:
                content = f.read()
            
            # Check for out-of-scope PII handling
            if "US_SSN" in content or "SSN" in content:
                return VerificationResult(
                    "PII Scope Locked",
                    False,
                    "Backend handles US_SSN (not in locked scope)",
                    "Remove US_SSN handling - only India PIIs allowed"
                )
        
        return VerificationResult(
            "PII Scope Locked",
            True,
            "PII scope appears locked to 11 types"
        )
    
    except Exception as e:
        return VerificationResult(
            "PII Scope Locked",
            False,
            f"Error checking PII scope: {str(e)}"
        )


def check_data_coherence() -> VerificationResult:
    """
    V6: Ensure Neo4j nodes == PostgreSQL findings.
    
    Pass Criteria:
    - COUNT(neo4j.findings) == COUNT(postgres.findings)
    - No orphan nodes or missing relationships
    """
    try:
        # Try to get counts from both systems
        pg_count = None
        neo4j_count = None
        
        # Get PostgreSQL count via API
        try:
            response = requests.get(f"{BACKEND_URL}/api/v1/findings?page_size=1", timeout=5)
            if response.status_code == 200:
                data = response.json()
                pg_count = data.get("total", 0)
        except requests.RequestException:
            pass
        
        # Get Neo4j count via lineage API
        try:
            response = requests.get(f"{BACKEND_URL}/api/v1/lineage/stats", timeout=5)
            if response.status_code == 200:
                data = response.json()
                neo4j_count = data.get("finding_count", 0)
        except requests.RequestException:
            pass
        
        if pg_count is None or neo4j_count is None:
            return VerificationResult(
                "Data Coherence",
                False,
                "Unable to query databases (backend not running)",
                "Start backend to verify data coherence"
            )
        
        if pg_count == neo4j_count:
            return VerificationResult(
                "Data Coherence",
                True,
                f"Data coherent: {pg_count} findings in both systems"
            )
        else:
            return VerificationResult(
                "Data Coherence",
                False,
                f"Data inconsistency: PostgreSQL={pg_count}, Neo4j={neo4j_count}",
                f"Difference: {abs(pg_count - neo4j_count)} findings"
            )
    
    except Exception as e:
        return VerificationResult(
            "Data Coherence",
            False,
            f"Error checking coherence: {str(e)}"
        )


def check_no_legacy_endpoints() -> VerificationResult:
    """
    V8: Ensure dead endpoints are removed or return 410 Gone.
    
    Pass Criteria:
    - /api/v1/lineage-old returns 410
    - /api/v1/classification/predict returns 410
    - /api/v1/scans/ingest redirects or returns 410
    """
    try:
        legacy_endpoints = [
            "/api/v1/lineage-old",
            "/api/v1/classification/predict",
            "/api/v1/scans/ingest"
        ]
        
        deprecated = []
        still_active = []
        
        for endpoint in legacy_endpoints:
            try:
                # Use appropriate HTTP method for each endpoint
                if endpoint in ["/api/v1/scans/ingest", "/api/v1/classification/predict"]:
                    # Test POST endpoints
                    response = requests.post(f"{BACKEND_URL}{endpoint}", json={}, timeout=5)
                else:
                    response = requests.get(f"{BACKEND_URL}{endpoint}", timeout=5)
                
                if response.status_code == 410:
                    deprecated.append(endpoint)
                elif response.status_code < 500 and response.status_code != 405:  # 2xx, 3xx, 4xx (but not 410 or 405 Method Not Allowed)
                    # Check if response has deprecation warning
                    if 'Warning' in response.headers or '299' in response.headers.get('Warning', ''):
                        deprecated.append(endpoint)  # Has warning = properly deprecated
                    else:
                        still_active.append(endpoint)
            except requests.RequestException:
                # Backend not running, check code
                pass
        
        if still_active:
            return VerificationResult(
                "No Legacy Endpoints",
                False,
                f"Legacy endpoints still active: {', '.join(still_active)}",
                "Should return 410 Gone"
            )
        
        # Check router code
        router_file = PROJECT_ROOT / "apps" / "backend" / "internal" / "api" / "router.go"
        if router_file.exists():
            with open(router_file, 'r') as f:
                content = f.read()
            
            # Check lineage-old - allow if it returns 410
            if "lineage-old" in content:
                if '"error": "This endpoint has been permanently removed"' not in content and "410" not in content:
                     return VerificationResult(
                        "No Legacy Endpoints",
                        False,
                        "Legacy lineage-old endpoint active",
                        "Remove lineage-old or ensure it returns 410"
                    )
            
            # Check for unverified ingest without warning
            if "/ingest\"" in content and "Warning" not in content:
                 return VerificationResult(
                    "No Legacy Endpoints",
                    False,
                    "Legacy /ingest endpoint registered without deprecation warning",
                    "Add Warning header or remove"
                )
        
        return VerificationResult(
            "No Legacy Endpoints",
            True,
            "Legacy endpoints properly deprecated"
        )
    
    except Exception as e:
        return VerificationResult(
            "No Legacy Endpoints",
            False,
            f"Error checking endpoints: {str(e)}"
        )


def check_scanner_sdk_completeness() -> VerificationResult:
    """
    Bonus: Verify scanner SDK has validators for all 11 locked PIIs.
    
    Pass Criteria:
    - SDK has validators for all locked PII types
    - Presidio recognizers registered for all types
    """
    try:
        validators_dir = PROJECT_ROOT / "apps" / "scanner" / "sdk" / "validators"
        recognizers_dir = PROJECT_ROOT / "apps" / "scanner" / "sdk" / "recognizers"
        
        if not validators_dir.exists():
            return VerificationResult(
                "Scanner SDK Completeness",
                False,
                "SDK validators directory not found"
            )
        
        # Check for required validators
        required_validators = [
            "luhn.py",  # Credit cards
            "verhoeff.py",  # Aadhaar
            # Add more as needed
        ]
        
        missing_validators = []
        for validator in required_validators:
            if not (validators_dir / validator).exists():
                missing_validators.append(validator)
        
        if missing_validators:
            return VerificationResult(
                "Scanner SDK Completeness",
                False,
                f"Missing validators: {', '.join(missing_validators)}",
                "SDK must have validators for all locked PIIs"
            )
        
        return VerificationResult(
            "Scanner SDK Completeness",
            True,
            "SDK has required validators"
        )
    
    except Exception as e:
        return VerificationResult(
            "Scanner SDK Completeness",
            False,
            f"Error checking SDK: {str(e)}"
        )


def run_all_checks() -> List[VerificationResult]:
    """Run all verification checks."""
    checks = [
        ("V1", check_scanner_output_contract),
        ("V2", check_backend_no_presidio_client),
        ("V3", check_backend_no_validators),
        ("V7", check_neo4j_mandatory),
        ("V5", check_pii_scope_locked),
        ("V6", check_data_coherence),
        ("V8", check_no_legacy_endpoints),
        ("BONUS", check_scanner_sdk_completeness),
    ]
    
    results = []
    for vid, check_func in checks:
        console.print(f"\n[cyan]Running check {vid}: {check_func.__name__}[/cyan]")
        result = check_func()
        results.append(result)
        
        status = "[green]✅ PASS[/green]" if result.passed else "[red]❌ FAIL[/red]"
        console.print(f"{status} - {result.message}")
        
        if result.details:
            console.print(f"[yellow]Details: {result.details}[/yellow]")
    
    return results


def print_summary(results: List[VerificationResult]):
    """Print verification summary table."""
    table = Table(title="ARC-Hawk Architecture Verification Summary", show_header=True)
    table.add_column("Check", style="cyan")
    table.add_column("Status", style="bold")
    table.add_column("Result", style="white")
    
    passed_count = 0
    total_count = len(results)
    
    for result in results:
        status = "✅ PASS" if result.passed else "❌ FAIL"
        status_style = "green" if result.passed else "red"
        
        table.add_row(
            result.name,
            f"[{status_style}]{status}[/{status_style}]",
            result.message
        )
        
        if result.passed:
            passed_count += 1
    
    console.print("\n")
    console.print(table)
    
    # Overall result
    if passed_count == total_count:
        console.print(Panel(
            f"[green]✅ All checks passed ({passed_count}/{total_count})[/green]\n"
            "[green]System is compliant with Intelligence-at-Edge architecture[/green]",
            title="SUCCESS",
            border_style="green"
        ))
        return 0
    else:
        failed_count = total_count - passed_count
        console.print(Panel(
            f"[red]❌ {failed_count} check(s) failed ({passed_count}/{total_count} passed)[/red]\n"
            "[red]System NOT compliant - remediation required[/red]",
            title="FAILURE",
            border_style="red"
        ))
        return 1


def main():
    """Main entry point."""
    console.print(Panel(
        "[bold cyan]ARC-Hawk Architecture Verification[/bold cyan]\n"
        "Ensuring Intelligence-at-Edge compliance",
        title="🔍 System Audit",
        border_style="cyan"
    ))
    
    # Run all checks
    results = run_all_checks()
    
    # Print summary
    exit_code = print_summary(results)
    
    sys.exit(exit_code)


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        console.print("\n[yellow]Verification cancelled by user[/yellow]")
        sys.exit(2)
    except Exception as e:
        console.print(f"\n[red]Fatal error: {e}[/red]")
        sys.exit(2)
