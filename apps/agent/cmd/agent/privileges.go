//go:build !windows

package main

import (
	"fmt"
	"os"
)

func checkPrivileges() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("hawk-agent must run as root (current euid=%d); use sudo or run as a system service", os.Geteuid())
	}
	return nil
}
