package databases

import "fmt"

// cfgString returns the first non-empty string value for any of the given
// keys in the config map. JSON numbers unmarshal as float64, which fmt.Sprintf
// renders with trailing ".000000" — coerce those to integer text for port
// and similar fields by trimming the fractional part when it's zero.
func cfgString(config map[string]any, keys ...string) string {
	for _, k := range keys {
		v, ok := config[k]
		if !ok || v == nil {
			continue
		}
		switch t := v.(type) {
		case string:
			if t != "" {
				return t
			}
		case float64:
			if t == float64(int64(t)) {
				return fmt.Sprintf("%d", int64(t))
			}
			return fmt.Sprintf("%v", t)
		case int:
			return fmt.Sprintf("%d", t)
		case int64:
			return fmt.Sprintf("%d", t)
		default:
			s := fmt.Sprintf("%v", t)
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}
