//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

func checkPrivileges() error {
	modadvapi32 := syscall.NewLazyDLL("advapi32.dll")
	procOpenProcessToken := modadvapi32.NewProc("OpenProcessToken")
	procGetTokenInformation := modadvapi32.NewProc("GetTokenInformation")

	// Get current process token.
	var token syscall.Token
	handle, err := syscall.GetCurrentProcess()
	if err != nil {
		return fmt.Errorf("get current process: %w", err)
	}

	const tokenQuery = 0x0008
	r1, _, e1 := procOpenProcessToken.Call(
		uintptr(handle),
		uintptr(tokenQuery),
		uintptr(unsafe.Pointer(&token)),
	)
	if r1 == 0 {
		return fmt.Errorf("cannot open process token: %v", e1)
	}
	defer token.Close()

	// Query token elevation.
	// TokenElevation = 20
	const tokenElevation = 20
	var elevation struct {
		TokenIsElevated uint32
	}
	var retLen uint32

	r1, _, e1 = procGetTokenInformation.Call(
		uintptr(token),
		uintptr(tokenElevation),
		uintptr(unsafe.Pointer(&elevation)),
		uintptr(unsafe.Sizeof(elevation)),
		uintptr(unsafe.Pointer(&retLen)),
	)
	if r1 == 0 {
		return fmt.Errorf("cannot query token elevation: %v", e1)
	}

	if elevation.TokenIsElevated == 0 {
		return fmt.Errorf("hawk-agent must run as Administrator; right-click and select 'Run as administrator'")
	}
	return nil
}
