package utils

import (
	"encoding/json"
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

var sensitiveKeys = map[string]bool{
	"password": true, "token": true, "secret": true, "key": true,
	"credential": true, "api_key": true, "aadhaar": true, "pan": true,
	"normalized_match": true, "authorization": true, "jwt": true,
	"refresh_token": true, "access_token": true, "encryption_key": true,
}

func ScrubJSONLog(jsonStr string) string {
	var obj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return "[invalid json]"
	}
	scrubbed := scrubValue(obj)
	out, err := json.Marshal(scrubbed)
	if err != nil {
		return "[marshal error]"
	}
	return string(out)
}

func scrubValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		for k := range val {
			if sensitiveKeys[strings.ToLower(k)] {
				val[k] = "[REDACTED]"
			} else {
				val[k] = scrubValue(val[k])
			}
		}
		return val
	case []interface{}:
		for i, item := range val {
			val[i] = scrubValue(item)
		}
		return val
	default:
		return v
	}
}

// SanitizeForLog is the single call-site for safe logging of values that may
// contain PII (sample text, user input, DB values). It applies DefaultScrubConfig
// so callers don't need to manage configuration.
//
// Usage:
//   log.Printf("finding sample: %s", utils.SanitizeForLog(finding.SampleText))
func SanitizeForLog(s string) string {
	return ScrubPII(s, &DefaultScrubConfig)
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
