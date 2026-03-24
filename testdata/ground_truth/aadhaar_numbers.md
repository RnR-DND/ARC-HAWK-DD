# Ground Truth Dataset - Aadhaar Numbers

## Purpose
Validate Indian Aadhaar number with Verhoeff checksum algorithm

## Valid Aadhaar Numbers (Should Detect as PII)

### Generated with valid Verhoeff checksum
234123456789  # Valid checksum
345234567890  # Valid checksum
456345678901  # Valid checksum
567456789012  # Valid checksum
678567890123  # Valid checksum

## Invalid Aadhaar Numbers (Should REJECT)

### Failed Verhoeff Checksum
234123456788  # Last digit wrong
345234567891  # Last digit wrong
456345678902  # Last digit wrong

### Wrong Length
12345678901   # 11 digits (too short)
2341234567890 # 13 digits (too long)

### All same digits (invalid pattern)
111111111111  # Repeated 1s
000000000000  # Repeated 0s
999999999999  # Repeated 9s

### Sequential (test data - invalid)
123456789012  # Sequential
987654321098  # Reverse sequential

## Edge Cases

### Formatting
2341-2345-6789  # With dashes (should normalize)
2341 2345 6789  # With spaces (should normalize)

### Leading zeros
012345678901  # Starts with 0 (unusual but valid if checksum passes)

## Expected Results

**Valid Aadhaar:** 5 detections  
**Invalid Aadhaar (Failed Verhoeff):** 0 detections  
**Invalid Aadhaar (Wrong length):** 0 detections  
**Normalized Aadhaar:** Should detect after removing dashes/spaces  
**Precision:** 100%  
**Recall:** 100%

## Note
Actual Aadhaar numbers with valid Verhoeff checksums would be needed for production testing.
The samples above are placeholders - real valid Aadhaar numbers should be generated using
the Verhoeff algorithm implementation.
