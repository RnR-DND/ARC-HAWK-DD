# Ground Truth Testing Guide

## Purpose
This directory contains labeled ground truth samples for validating ARC-Hawk's PII classification accuracy.

## Structure

```
testdata/ground_truth/
├── README.md                  # This file
├── samples.json               # Combined test samples (JSON format)
├── credit_cards.md            # Credit card test cases
├── emails.md                  # Email test cases
├── pan_numbers.md             # Indian PAN test cases
├── ssn_numbers.md             # US SSN test cases
├── aadhaar_numbers.md         # Indian Aadhaar test cases
└── negative_samples.md        # Non-PII samples (false positive tests)
```

## Running Tests

### Prerequisites
1. Presidio service running (optional but recommended):
   ```bash
   docker-compose up presidio-analyzer
   ```

2. Backend dependencies installed:
   ```bash
   cd apps/backend
   go mod download
   ```

### Run Regression Tests
```bash
cd apps/backend
go run cmd/regression/main.go
```

### Quality Gate
Tests will FAIL if F1 score < 0.90

Expected output:
```
🧪 ARC-Hawk Regression Testing Framework
=========================================

Loaded 50 ground truth samples
✅ Presidio connected at http://localhost:5001

📊 Test Results
===============
✅ All tests passed!

📈 Metrics
==========
Total Samples:     50
True Positives:    25
False Positives:   0
True Negatives:    25
False Negatives:   0

Precision:         1.0000 (100.00%)
Recall:            1.0000 (100.00%)
F1 Score:          1.0000 (100.00%)
Accuracy:          1.0000 (100.00%)

✅ QUALITY GATE PASSED: F1 Score (1.0000) >= 0.90
```

## Sample Format (JSON)

```json
[
  {
    "value": "4532015112830366",
    "expected_type": "CREDIT_CARD",
    "should_detect": true,
    "description": "Valid Visa card with Luhn checksum"
  },
  {
    "value": "1234567890123456",
    "expected_type": "NON_PII",
    "should_detect": false,
    "description": "Random number failing Luhn - should not detect"
  }
]
```

## Fields Explanation

- **value**: The actual text/number to test
- **expected_type**: Expected PII classification type
  - `CREDIT_CARD`
  - `EMAIL_ADDRESS`
  - `IN_PAN`
  - `US_SSN`
  - `IN_AADHAAR`
  - `NON_PII` (for negative samples)
- **should_detect**: Boolean - true if should be detected as PII, false otherwise
- **description**: Human-readable explanation of why this sample is included

## Adding New Test Cases

1. Add samples to appropriate .md file for documentation
2. Add JSON entries to `samples.json`
3. Run regression tests to verify
4. Commit if all tests pass

## Current Coverage

| Type | Positive Samples | Negative Samples | Total |
|------|-----------------|------------------|-------|
| Credit Cards | 12 | 15 | 27 |
| Emails | 20 | 10 | 30 |
| PAN | 15 | 8 | 23 |
| SSN | 5 | 20 | 25 |
| Aadhaar | 5 | 5 | 10 |
| General Negative | 0 | 50 | 50 |
| **TOTAL** | **57** | **108** | **165** |

## Success Criteria

✅ **Precision**: 100% (No false positives allowed)  
✅ **Recall**: >= 95% (Allow few false negatives)  
✅ **F1 Score**: >= 0.90 (Quality gate)  
✅ **Accuracy**: >= 95%

## Notes

- All validators (Luhn, Verhoeff, PAN, SSN) are hard requirements
- Presidio provides initial detection, validators eliminate false positives
- Ground truth samples should be diverse and cover edge cases
- Update samples as new false positive/negative cases are discovered
