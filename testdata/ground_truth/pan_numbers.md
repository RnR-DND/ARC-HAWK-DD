# Ground Truth Dataset - Indian PAN

## Purpose
Validate Indian PAN (Permanent Account Number) format validation

## Valid PAN Numbers (Should Detect as PII)

### Individual (P)
ABCDE1234F
PQRST5678K
UVWXY9012L

### Company (C)
ABCDE1234C
XYZKL5678C
MNOPQ9012C

### Hindu Undivided Family (H)
ABCDE1234H
PQRST5678H

### Firm/Partnership (F)
ABCDE1234F
LMNOP5678F

### Trust (T)
ABCDE1234T
XYZKL5678T

## Invalid PAN Numbers (Should REJECT)

### Wrong length
ABCDE1234    # Too short (9 chars)
ABCDE12345F  # Too long (11 chars)

### Wrong format
12345ABCDF   # Numbers first (should be letters)
ABCDE123F4   # Letter in wrong position
abcde1234f   # Lowercase (should be uppercase)

### Invalid structure
ABCD11234F   # Only 4 letters at start (need 5)
ABCDE12345   # No letter at end

### Invalid characters
ABCDE-1234F  # Contains hyphen
ABCDE 1234F  # Contains space

## Edge Cases

### Case sensitivity
ABCDE1234F  # Uppercase (valid)
abcde1234f  # Lowercase (should normalize to uppercase)
AbCdE1234F  # Mixed case (should normalize)

### Fourth character entity type
ABCPP1234F  # Individual
ABCPC1234C  # Company
ABCPH1234H  # HUF
ABCPF1234F  # Firm
ABCPT1234T  # Trust
ABCPG1234G  # Government
ABCPL1234L  # Local Authority

## Expected Results

**Valid PANs:** 15+ detections  
**Invalid PANs:** 0 detections  
**Precision:** 100%  
**Recall:** 100%  
**Entity Type Detection:** Accurate for all valid PANs
