package classifier

import (
	"crypto/sha256"
	"fmt"
)

// Dedup removes duplicate findings keyed by (pii_type, source_path, value_hash_prefix).
func Dedup(findings []ClassifiedFinding) []ClassifiedFinding {
	seen := make(map[string]struct{})
	out := make([]ClassifiedFinding, 0, len(findings))
	for _, f := range findings {
		h := sha256.Sum256([]byte(f.ValueHash))
		key := fmt.Sprintf("%s|%s|%x", f.PIIType, f.SourcePath, h[:])
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			out = append(out, f)
		}
	}
	return out
}
