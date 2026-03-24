package utils

import (
	"regexp"
	"strings"
)

var (
	emailPattern      = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	phonePattern      = regexp.MustCompile(`(?:\+?[91][-\s]?)?[6-9][0-9]{9}`)
	aadhaarPattern    = regexp.MustCompile(`[2-9]{1}[0-9]{3}[0-9]{4}[0-9]{4}`)
	panPattern        = regexp.MustCompile(`[A-Z]{5}[0-9]{4}[A-Z]`)
	creditCardPattern = regexp.MustCompile(`[0-9]{4}[-\s]?[0-9]{4}[-\s]?[0-9]{4}[-\s]?[0-9]{4}`)
	upiPattern        = regexp.MustCompile(`[a-zA-Z0-9._-]+@[a-zA-Z0-9._-]+`)
	ifscPattern       = regexp.MustCompile(`[A-Z]{4}0[A-Z0-9]{6}`)

	ipv4Pattern = regexp.MustCompile(`(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)`)
)

type ScrubConfig struct {
	ScrubEmail      bool
	ScrubPhone      bool
	ScrubAadhaar    bool
	ScrubPAN        bool
	ScrubCreditCard bool
	ScrubUPI        bool
	ScrubIFSC       bool
	ScrubIP         bool
	ScrubPasswords  bool
}

var DefaultScrubConfig = ScrubConfig{
	ScrubEmail:      true,
	ScrubPhone:      true,
	ScrubAadhaar:    true,
	ScrubPAN:        true,
	ScrubCreditCard: true,
	ScrubUPI:        true,
	ScrubIFSC:       true,
	ScrubIP:         true,
	ScrubPasswords:  true,
}

func ScrubPII(input string, config *ScrubConfig) string {
	if config == nil {
		config = &DefaultScrubConfig
	}

	result := input

	if config.ScrubEmail {
		result = emailPattern.ReplaceAllString(result, "[EMAIL_REDACTED]")
	}

	if config.ScrubPhone {
		result = phonePattern.ReplaceAllString(result, "[PHONE_REDACTED]")
	}

	if config.ScrubAadhaar {
		result = aadhaarPattern.ReplaceAllString(result, "[AADHAAR_REDACTED]")
	}

	if config.ScrubPAN {
		result = panPattern.ReplaceAllString(result, "[PAN_REDACTED]")
	}

	if config.ScrubCreditCard {
		result = creditCardPattern.ReplaceAllString(result, "[CREDIT_CARD_REDACTED]")
	}

	if config.ScrubUPI {
		result = upiPattern.ReplaceAllString(result, "[UPI_REDACTED]")
	}

	if config.ScrubIFSC {
		result = ifscPattern.ReplaceAllString(result, "[IFSC_REDACTED]")
	}

	if config.ScrubIP {
		result = ipv4Pattern.ReplaceAllString(result, "[IP_REDACTED]")
	}

	if config.ScrubPasswords {
		passwordPattern := regexp.MustCompile(`(?i)(password|passwd|pwd|secret|token|apikey|api_key|access_key|accesskey)["']?\s*[:=]\s*["']?([^\s"'\}]+)`)
		result = passwordPattern.ReplaceAllString(result, "$1: [REDACTED]")
	}

	return result
}

func ScrubJSONLog(jsonStr string) string {
	var result strings.Builder
	inString := false
	escapeNext := false
	currentKey := ""

	for i := 0; i < len(jsonStr); i++ {
		ch := jsonStr[i]

		if escapeNext {
			result.WriteByte(ch)
			escapeNext = false
			continue
		}

		if ch == '\\' {
			result.WriteByte(ch)
			escapeNext = true
			continue
		}

		if ch == '"' {
			if !inString {
				inString = true
				currentKey = ""
			} else {
				currentKey = result.String()
				keyEnd := strings.LastIndex(result.String(), "\"")
				if keyEnd > 0 {
					currentKey = result.String()[keyEnd+1:]
				}
				inString = false
			}
			result.WriteByte(ch)
			continue
		}

		if inString {
			result.WriteByte(ch)
			continue
		}

		if ch == ':' {
			trimmedKey := strings.TrimSpace(currentKey)
			if shouldScrubKey(trimmedKey) {
				result.WriteString(": [REDACTED]")
				i++
				for i < len(jsonStr) && (jsonStr[i] == ' ' || jsonStr[i] == '\t') {
					i++
				}
				if i < len(jsonStr) && jsonStr[i] == '"' {
					inString = true
					for i < len(jsonStr) && jsonStr[i] != '"' {
						i++
					}
					if i < len(jsonStr) {
						i--
					}
				} else if i < len(jsonStr) && (jsonStr[i] == '{' || jsonStr[i] == '[') {
					i--
				}
			} else {
				result.WriteByte(ch)
			}
			continue
		}

		result.WriteByte(ch)
	}

	return result.String()
}

func shouldScrubKey(key string) bool {
	lowerKey := strings.ToLower(key)
	sensitiveKeys := []string{
		"password", "passwd", "pwd", "secret", "token", "apikey", "api_key",
		"access_key", "accesskey", "private_key", "privatekey", "credential",
		"config", "connection_string", "connection_string", "db_password",
		"db_pass", "encryption_key", "jwt_secret", "admin_password",
	}
	for _, sk := range sensitiveKeys {
		if strings.Contains(lowerKey, sk) {
			return true
		}
	}
	return false
}

type LogMessage struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Timestamp string                 `json:"timestamp"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

func ScrubLogMessage(msg *LogMessage) *LogMessage {
	msg.Message = ScrubPII(msg.Message, nil)

	if msg.Fields != nil {
		for key, value := range msg.Fields {
			if strVal, ok := value.(string); ok {
				msg.Fields[key] = ScrubPII(strVal, nil)
			}
		}
	}

	return msg
}
