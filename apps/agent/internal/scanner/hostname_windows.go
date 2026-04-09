//go:build windows

package scanner

import "os"

func hostnameOS() (string, error) {
	return os.Hostname()
}
