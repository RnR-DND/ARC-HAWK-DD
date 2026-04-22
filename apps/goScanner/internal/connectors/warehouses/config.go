package warehouses

import "fmt"

// cfgInt returns an integer from cfg[key], clamped to [1, maxVal].
// Falls back to def when the key is absent, zero, or unparseable.
func cfgInt(cfg map[string]any, key string, def, maxVal int) int {
	v, ok := cfg[key]
	if !ok || v == nil {
		return def
	}
	var n int
	switch t := v.(type) {
	case float64:
		n = int(t)
	case int:
		n = t
	case int64:
		n = int(t)
	case string:
		fmt.Sscanf(t, "%d", &n) //nolint:errcheck
	}
	if n <= 0 {
		return def
	}
	if n > maxVal {
		return maxVal
	}
	return n
}
