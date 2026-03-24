"""
Unified Scanner with SDK Integration
=====================================
NEW FLOW:
1. Hawk-eye scanner finds potential matches (regex)
2. SDK validates with context + math
3. Only verified findings sent to backend

This replaces the old flow where backend did validation.
"""

import os
import sys
import json
import argparse
from pathlib import Path
from typing import List, Dict, Any

# Add scanner to path
sys.path.insert(0, str(Path(__file__).parent.parent / "apps" / "scanner"))

from sdk.engine import SharedAnalyzerEngine
from sdk.recognizers import AadhaarRecognizer, PANRecognizer, CreditCardRecognizer
from sdk.schema import VerifiedFinding, SourceInfo
from sdk.context_extractor import extract_context_from_file


def initialize_sdk_engine(config_path: str = None):
    """
    Initialize the SDK engine with custom recognizers.
    
    Returns:
        Configured AnalyzerEngine
    """
    print("[SDK] Initializing intelligence-at-edge scanner...")
    
    # Get engine (singleton)
    if not config_path:
        config_path = "apps/scanner/sdk/config.yml"
    
    engine = SharedAnalyzerEngine.get_engine(config_path)
    
    # Register custom recognizers
    print("[SDK] Registering custom recognizers...")
    SharedAnalyzerEngine.add_recognizer(AadhaarRecognizer())
    SharedAnalyzerEngine.add_recognizer(PANRecognizer())
    SharedAnalyzerEngine.add_recognizer(CreditCardRecognizer())
    
    print("[SDK] Scanner ready with mathematical validation enabled")
    return engine


def process_hawk_output(hawk_json_path: str, engine) -> List[VerifiedFinding]:
    """
    Process Hawk-eye scanner output through SDK validation.
    
    Args:
        hawk_json_path: Path to Hawk-eye JSON output
        engine: SharedAnalyzerEngine instance
        
    Returns:
        List of verified findings (validated by SDK)
    """
    print(f"\n[SDK] Processing Hawk-eye output: {hawk_json_path}")
    
    # Load Hawk-eye results
    with open(hawk_json_path, 'r') as f:
        hawk_data = json.load(f)
    
    verified_findings = []
    total_candidates = 0
    discarded = 0
    
    # Process filesystem findings
    if 'fs' in hawk_data:
        for finding in hawk_data['fs']:
            total_candidates += 1
            
            # Extract context window
            file_path = finding.get('file_path', '')
            line_num = finding.get('line_number', 1)
            
            if os.path.exists(file_path):
                context = extract_context_from_file(file_path, line_num, window_size=5)
            else:
                context = finding.get('match', '')
            
            # Run SDK validation
            results = engine.analyze(
                text=context,
                language='en',
                entities=["IN_AADHAAR", "IN_PAN", "CREDIT_CARD"]
            )
            
            if results:
                # SDK validated this finding
                for result in results:
                    source = SourceInfo(
                        path=file_path,
                        line=line_num,
                        data_source="filesystem"
                    )
                    
                    verified = VerifiedFinding.create_from_analysis(
                        presidio_result=result,
                        text=context,
                        source_info=source,
                        pattern_name=finding.get('pattern_name', 'Unknown'),
                        validators=["mathematical"]  # Since our recognizers use math
                    )
                    
                    verified_findings.append(verified)
                    
                print(f"  ✓ Verified: {result.entity_type} in {file_path}:{line_num}")
            else:
                # SDK rejected (failed validation)
                discarded += 1
                print(f"  ✗ Discarded: {finding.get('pattern_name')} in {file_path}:{line_num} (failed validation)")
    
    print(f"\n[SDK] Validation Summary:")
    print(f"  Total candidates: {total_candidates}")
    print(f"  Verified: {len(verified_findings)}")
    print(f"  Discarded: {discarded}")
    print(f"  Reduction: {discarded/total_candidates*100:.1f}%")
    
    return verified_findings


def send_to_backend(findings: List[VerifiedFinding], backend_url: str):
    """
    Send verified findings to backend ingestion API.
    
    Args:
        findings: List of VerifiedFinding objects
        backend_url: Backend API URL
    """
    import requests
    
    payload = {
        "verified_findings": [f.to_dict() for f in findings],
        "scanner_version": "2.0-sdk",
        "validation_enabled": True
    }
    
    print(f"\n[INGEST] Sending {len(findings)} verified findings to backend...")
    
    try:
        response = requests.post(
            f"{backend_url}/api/v1/scans/ingest-verified",
            json=payload,
            timeout=30
        )
        
        if response.status_code == 200:
            print(f"[INGEST] ✓ Success: {response.json()}")
        else:
            print(f"[INGEST] ✗ Error {response.status_code}: {response.text}")
    
    except Exception as e:
        print(f"[INGEST] ✗ Failed to send to backend: {e}")


def main():
    """Main entry point for unified scanner with SDK."""
    parser = argparse.ArgumentParser(description="ARC-Hawk Unified Scanner (SDK Mode)")
    parser.add_argument("--hawk-output", default="scan_output.json", help="Hawk-eye JSON output")
    parser.add_argument("--backend-url", default="http://localhost:8080", help="Backend API URL")
    parser.add_argument("--strict-mode", action="store_true", help="Use Phase 1 patterns only (11 PIIs)")
    parser.add_argument("--dry-run", action="store_true", help="Don't send to backend")
    
    args = parser.parse_args()
    
    # Initialize SDK with appropriate config
    if args.strict_mode:
        config = "apps/scanner/sdk/config.yml"
        print("[STRICT-MODE] Using Phase 1 patterns only (11 PIIs)")
    else:
        config = None
        print("[STANDARD-MODE] Using all configured patterns")
    
    engine = initialize_sdk_engine(config)
    
    # Process Hawk-eye output through SDK
    verified_findings = process_hawk_output(args.hawk_output, engine)
    
    # Send to backend (unless dry-run)
    if not args.dry_run and verified_findings:
        send_to_backend(verified_findings, args.backend_url)
    elif args.dry_run:
        print("\n[DRY-RUN] Skipping backend ingestion")
        print(f"\nSample verified finding:")
        if verified_findings:
            print(json.dumps(verified_findings[0].to_dict(), indent=2))
    
    print(f"\n[COMPLETE] Scan finished. {len(verified_findings)} verified findings ready.")


if __name__ == "__main__":
    main()
