# Ground Truth Dataset - US SSN

## Purpose
Validate US Social Security Number with SSA rules and blacklist

## Valid SSN Numbers (Should Detect as PII)

### Standard format (9 digits)
123456789
234567890
345678901
456789012
567890123

### Area number variations (001-665, 667-899)
001234567  # Lowest valid area
665123456  # Just below 666 (reserved)
667123456  # Just above 666
899123456  # Highest valid area

## Invalid SSN Numbers (Should REJECT)

### SSA Rule Violations

#### Area number = 000
000123456  # Invalid: area cannot be 000

#### Area number = 666
666123456  # Invalid: 666 is reserved

#### Area number = 900-999 (ITIN range)
900123456  # Invalid: 900-999 reserved for ITIN
987654321  # Invalid: starts with 9

#### Group number = 00
123001234  # Invalid: group cannot be 00

#### Serial number = 0000
123450000  # Invalid: serial cannot be 0000

### Blacklist (Common Test Values)
000000000  # All zeros
111111111  # Repeated digits
222222222
333333333
444444444
555555555
777777777
888888888
999999999
123456789  # Sequential (in blacklist)
987654321  # Reverse sequential

### Format Issues
12-345-6789  # With dashes (should normalize)
123 45 6789  # With spaces (should normalize)
12345678     # Too short
1234567890   # Too long

## Edge Cases

### Normalization
123-45-6789  # Should normalize to 123456789
123 45 6789  # Should normalize to 123456789

### Recently issued (post-2011 randomization)
# After June 25, 2011, SSNs are randomized
# Any number not in blacklist and passing rules is potentially valid

## Expected Results

**Valid SSNs:** 5 detections  
**Invalid SSNs (SSA rules):** 0 detections  
**Invalid SSNs (Blacklist):** 0 detections  
**Normalized SSNs:** Should detect after normalization  
**Precision:** 100%  
**Recall:** 100%
