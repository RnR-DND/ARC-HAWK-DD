package validators

import (
	"regexp"
	"strconv"
)

var gstRegex = regexp.MustCompile(`^[0-3][0-9][A-Z]{5}[0-9]{4}[A-Z][0-9A-Z]Z[0-9A-Z]$`)

// ValidateGST validates a GST number.
func ValidateGST(gst string) bool {
	if !gstRegex.MatchString(gst) {
		return false
	}
	stateCode, err := strconv.Atoi(gst[:2])
	if err != nil {
		return false
	}
	return stateCode >= 1 && stateCode <= 38
}
