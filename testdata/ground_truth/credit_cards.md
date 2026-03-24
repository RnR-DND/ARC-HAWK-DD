# Ground Truth Dataset - Credit Cards

## Purpose
Validate credit card classification with Luhn checksum validation

## Valid Credit Cards (Should Detect as PII)

### Visa
4532015112830366
4916338506082832
4024007198964305

### Mastercard
5425233430109903
2221000000000009
5555555555554444

### American Express
378282246310005
371449635398431
378734493671000

### Discover
6011111111111117
6011000990139424
6011601160116611

## Invalid Credit Cards (Should REJECT - Failed Luhn)

### Failed Luhn Checksum
4532015112830367  # Last digit wrong
5425233430109904  # Last digit wrong
378282246310006   # Last digit wrong
6011111111111118  # Last digit wrong

### Random 16-digit numbers (not credit cards)
1234567812345670  # Passes Luhn but not a real card
9876543210987654  # Doesn't pass Luhn
1111111111111111  # Doesn't pass Luhn
0000000000000000  # Invalid

## Edge Cases

### Too short
123456781234567   # 15 digits (Amex length but invalid)

### Too long
45320151128303661 # 17 digits

### Non-numeric
4532-0151-1283-0366  # With dashes (should be normalized)
4532 0151 1283 0366  # With spaces (should be normalized)

## Expected Results

**Valid Cards:** 12 detections  
**Invalid Cards:** 0 detections (validators should reject all)  
**Precision:** 100% (no false positives)  
**Recall:** 100% (all valid cards detected)
