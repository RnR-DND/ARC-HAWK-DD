"""
Scanner Integration Example
============================
Complete example showing how Presidio + Validators work together.

This demonstrates the full Intelligence-at-Edge workflow:
1. Presidio analyzes text (ML/NLP detection)
2. Validators verify findings (mathematical validation)
3. VerifiedFindings sent to backend
"""

from sdk.engine import SharedAnalyzerEngine
from sdk.schema import VerifiedFinding, SourceInfo
from sdk.validation_pipeline import filter_and_validate_results


def scan_text_with_validation(text: str, source_path: str = "/example/file.txt"):
    """
    Complete scan workflow with Presidio + Validation.
    
    Args:
        text: Text to scan for PIIs
        source_path: Path to source file
        
    Returns:
        List of VerifiedFinding objects (validated PIIs only)
    """
    print(f"\n{'='*70}")
    print(f"🔍 Scanning text for PIIs (Intelligence-at-Edge)")
    print(f"{'='*70}\n")
    
    # Step 1: Get Presidio engine (with our custom recognizers)
    print("Step 1: Initializing Presidio AnalyzerEngine...")
    engine = SharedAnalyzerEngine.get_engine()
    print("✅ Presidio ready with 11 custom recognizers\n")
    
    # Step 2: Run Presidio analysis (ML/NLP detection)
    print("Step 2: Running Presidio ML/NLP analysis...")
    presidio_results = engine.analyze(
        text=text,
        language='en',
        # entities=None means analyze for ALL registered entities
    )
    print(f"✅ Presidio found {len(presidio_results)} potential PIIs\n")
    
    # Print Presidio results
    if presidio_results:
        print("Presidio Detections:")
        for i, result in enumerate(presidio_results, 1):
            matched_text = text[result.start:result.end]
            print(f"  {i}. {result.entity_type}: {matched_text}")
            print(f"     ML Confidence: {result.score:.2f}")
            print(f"     Position: {result.start}-{result.end}\n")
    
    # Step 3: Validate through pipeline (mathematical verification)
    print("\nStep 3: Running mathematical validation...")
    print("-" * 70)
    
    source_info = SourceInfo(
        path=source_path,
        line=None,
        column=None,
        data_source="filesystem",
        host="localhost"
    )
    
    verified_findings = filter_and_validate_results(
        presidio_results=presidio_results,
        text=text,
        source_info=source_info,
        pattern_name="presidio_ml"
    )
    
    print("-" * 70)
    print(f"\n✅ Final Result: {len(verified_findings)} VerifiedFindings")
    print(f"   (Rejected: {len(presidio_results) - len(verified_findings)} invalid PIIs)\n")
    
    # Print verified findings
    if verified_findings:
        print("Verified Findings (ready for backend):")
        for i, finding in enumerate(verified_findings, 1):
            print(f"  {i}. {finding.pii_type}")
            print(f"     Validators Passed: {finding.validators_passed}")
            print(f"     ML Confidence: {finding.ml_confidence:.2f}")
            print(f"     Hash: {finding.value_hash[:16]}...")
            print()
    
    return verified_findings


# Example usage
if __name__ == "__main__":
    # Example text with multiple PII types
    test_text = """
    Customer Record:
    Name: Rajesh Kumar
    Aadhaar: 2345 6789 0124
    PAN: AAAPC1234D
    Phone: +91 9876543210
    Email: rajesh.kumar@example.com
    Credit Card: 4532 0151 1283 0366
    UPI ID: rajesh@paytm
    IFSC: SBIN0001234
    Passport: A1234567
    """
    
    print("Test Text:")
    print(test_text)
    
    verified = scan_text_with_validation(test_text)
    
    print(f"\n{'='*70}")
    print(f"Summary: {len(verified)} PII(s) validated and ready for ingestion")
    print(f"{'='*70}")
