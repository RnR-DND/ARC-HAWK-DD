package validators

import "regexp"

var dlRegex = regexp.MustCompile(`^[A-Z]{2}[- ]?\d{2}[- ]?\d{4,7}$`)
var validDLStates = map[string]bool{
	"AP": true, "AR": true, "AS": true, "BR": true, "CG": true, "GA": true,
	"GJ": true, "HR": true, "HP": true, "JH": true, "KA": true, "KL": true,
	"MP": true, "MH": true, "MN": true, "ML": true, "MZ": true, "NL": true,
	"OD": true, "PB": true, "RJ": true, "SK": true, "TN": true, "TS": true,
	"TR": true, "UP": true, "UK": true, "WB": true, "AN": true, "CH": true,
	"DD": true, "DL": true, "JK": true, "LA": true, "LD": true, "PY": true,
}

// ValidateDrivingLicense validates an Indian driving license.
func ValidateDrivingLicense(dl string) bool {
	if !dlRegex.MatchString(dl) {
		return false
	}
	if len(dl) < 2 {
		return false
	}
	return validDLStates[dl[:2]]
}
