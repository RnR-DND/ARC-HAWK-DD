# Ground Truth Dataset - Email Addresses

## Purpose
Validate email address classification and format validation

## Valid Email Addresses (Should Detect as PII)

### Standard formats
john.doe@example.com
jane_smith@company.org
user123@domain.co.uk
contact@sub.domain.com
info@example.io

### Edge cases (valid)
user+tag@gmail.com
firstname.lastname@example.com
email@subdomain.example.com
1234567890@example.com
_user@example.com

### International domains
user@münchen.de
contact@zürich.ch
info@пример.рф

## Invalid Email Addresses (Should REJECT)

### Missing @
userdomain.com
john.doe.example.com

### Missing domain
user@
@example.com
user@.com

### Invalid characters
user name@example.com  # Space in local part
user@domain space.com  # Space in domain
user@@example.com      # Double @
user@exam ple.com      # Space in domain

### Missing TLD
user@domain
user@localhost

## Edge Cases

### Unusual but valid
very.unusual.@.unusual.com@example.com  # Valid per RFC
"user name"@example.com                  # Quoted local part (valid)

### Normalization test
john.doe@gmail.com
johndoe@gmail.com
# Gmail treats these as identical (removes dots before @)

## Expected Results

**Valid Emails:** 20+ detections  
**Invalid Emails:** 0 detections  
**Precision:** 100%  
**Recall:** 100%
