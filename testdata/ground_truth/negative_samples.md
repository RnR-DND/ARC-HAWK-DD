# Ground Truth Dataset - Negative Samples (Non-PII)

## Purpose
Validate that non-PII data is correctly rejected (no false positives)

## Random Numbers (Should NOT Detect)

### 16-digit numbers (look like credit cards but fail Luhn)
1234567890123456
9876543210987654
1111222233334444
5555666677778888
0000111122223333

### 12-digit numbers (look like Aadhaar but fail Verhoeff)
123456789123
987654321987
111111111111
000000000000

### 9-digit numbers (look like SSN but violate rules)
000000000  # Area = 000
666000000  # Area = 666
123000000  # Group = 00
123450000  # Serial = 0000

## Text that looks like PII but isn't

### Fake emails (invalid format)
user.domain.com
@example.com
user@
username@

### Fake phone numbers
123-456  # Too short
1234567890123456  # Too long
(000) 000-0000  # Invalid area code

### Random strings with numbers
ABC12345  # Too short for PAN
1234567   # Too short for any PII type
XYZ-123-ABC  # Not matching any pattern

## Common false positives to avoid

### Configuration values
DB_PORT=5432
API_KEY=1234567890abcdef
TIMEOUT=30000

### Dates formatted as numbers
20231225  # YYYYMMDD
01012024  # DDMMYYYY
12312023  # MMDDYYYY

### Order/Transaction IDs
ORD-123456789
TXN-987654321
INV-2023-12345

### Account numbers (internal, not PII)
ACC000123456
CUST999888777

### Version numbers
1.2.3.456
2.0.1234

### IP addresses (not PII)
192.168.1.1
10.0.0.255
172.16.254.1

### MAC addresses (not PII)
00:1B:44:11:3A:B7
A4-5E-60-E7-93-9B

## Product codes / SKUs
SKU-123456789012
PROD-ABC-123-XYZ
UPC-012345678905  # Has checksum but not a credit card

## Expected Results

**Detections:** 0 (all should be rejected)  
**False Positives:** 0  
**Precision Impact:** Critical - these MUST NOT be detected as PII
