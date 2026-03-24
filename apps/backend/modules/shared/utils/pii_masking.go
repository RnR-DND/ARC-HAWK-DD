package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

type PIIMasker struct {
	replacements map[string]string
}

func NewPIIMasker() *PIIMasker {
	return &PIIMasker{
		replacements: make(map[string]string),
	}
}

func (m *PIIMasker) MaskValue(value string) string {
	if value == "" {
		return ""
	}

	masked := value
	for pattern, replacement := range m.replacements {
		masked = strings.ReplaceAll(masked, pattern, replacement)
	}

	return masked
}

func (m *PIIMasker) HashValue(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

func (m *PIIMasker) MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return m.HashValue(email)
	}

	localPart := parts[0]
	domain := parts[1]

	if len(localPart) <= 2 {
		localPart = strings.Repeat("*", len(localPart))
	} else {
		localPart = localPart[:2] + strings.Repeat("*", len(localPart)-2)
	}

	return fmt.Sprintf("%s@%s", localPart, domain)
}

func (m *PIIMasker) MaskPhone(phone string) string {
	if len(phone) <= 4 {
		return strings.Repeat("*", len(phone))
	}
	return strings.Repeat("*", len(phone)-4) + phone[len(phone)-4:]
}

func (m *PIIMasker) MaskAadhaar(aadhaar string) string {
	if len(aadhaar) != 12 {
		return aadhaar
	}
	return "XXXX-XXXX-" + aadhaar[len(aadhaar)-4:]
}

func (m *PIIMasker) MaskPAN(pan string) string {
	if len(pan) != 10 {
		return pan
	}
	return strings.ToUpper(pan[:5] + "XXXX" + pan[9:])
}

func (m *PIIMasker) MaskCreditCard(cc string) string {
	cc = regexp.MustCompile(`[^0-9]`).ReplaceAllString(cc, "")
	if len(cc) < 4 {
		return strings.Repeat("*", len(cc))
	}
	return "****-****-****-" + cc[len(cc)-4:]
}

func (m *PIIMasker) MaskUPI(upi string) string {
	parts := strings.Split(upi, "@")
	if len(parts) != 2 {
		return m.HashValue(upi)
	}

	userPart := parts[0]
	if len(userPart) <= 2 {
		userPart = strings.Repeat("*", len(userPart))
	} else {
		userPart = userPart[:2] + strings.Repeat("*", len(userPart)-2)
	}

	return fmt.Sprintf("%s@%s", userPart, parts[1])
}

func (m *PIIMasker) MaskIFSC(ifsc string) string {
	if len(ifsc) != 11 {
		return ifsc
	}
	return ifsc[:4] + "XXXXXXX"
}

func (m *PIIMasker) MaskIPAddress(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ip
	}
	return fmt.Sprintf("%s.%s.%s.*", parts[0], parts[1], parts[2])
}

func (m *PIIMasker) MaskName(name string) string {
	words := strings.Fields(name)
	for i, word := range words {
		if len(word) <= 2 {
			continue
		}
		words[i] = word[:1] + strings.Repeat("*", len(word)-1)
	}
	return strings.Join(words, " ")
}

func (m *PIIMasker) MaskDateOfBirth(dob string) string {
	return "****-**-**"
}

func (m *PIIMasker) MaskSSN(ssn string) string {
	if len(ssn) != 9 {
		return ssn
	}
	return "***-**-" + ssn[len(ssn)-4:]
}

func (m *PIIMasker) MaskPassport(passport string) string {
	if len(passport) < 4 {
		return strings.Repeat("*", len(passport))
	}
	return strings.Repeat("*", len(passport)-4) + passport[len(passport)-4:]
}

func (m *PIIMasker) MaskDrivingLicense(license string) string {
	if len(license) <= 4 {
		return strings.Repeat("*", len(license))
	}
	return strings.Repeat("*", len(license)-4) + license[len(license)-4:]
}

func (m *PIIMasker) MaskVoterID(voterID string) string {
	if len(voterID) <= 4 {
		return strings.Repeat("*", len(voterID))
	}
	return strings.Repeat("*", len(voterID)-4) + voterID[len(voterID)-4:]
}

func (m *PIIMasker) MaskBankAccount(account string) string {
	if len(account) <= 4 {
		return strings.Repeat("*", len(account))
	}
	return strings.Repeat("*", len(account)-4) + account[len(account)-4:]
}
