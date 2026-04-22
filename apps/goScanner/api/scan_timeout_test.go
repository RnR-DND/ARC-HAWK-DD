package api

import (
	"testing"
	"time"
)

func TestResolveScanTimeout_DefaultWhenUnset(t *testing.T) {
	t.Setenv("SCAN_TIMEOUT_SECONDS", "")
	if got := resolveScanTimeout(); got != defaultScanTimeout {
		t.Errorf("unset: got %v, want %v", got, defaultScanTimeout)
	}
}

func TestResolveScanTimeout_ParsesOverride(t *testing.T) {
	t.Setenv("SCAN_TIMEOUT_SECONDS", "90")
	if got := resolveScanTimeout(); got != 90*time.Second {
		t.Errorf("override 90: got %v, want 90s", got)
	}
}

func TestResolveScanTimeout_RejectsNonNumeric(t *testing.T) {
	t.Setenv("SCAN_TIMEOUT_SECONDS", "not-a-number")
	if got := resolveScanTimeout(); got != defaultScanTimeout {
		t.Errorf("garbage: got %v, want default %v", got, defaultScanTimeout)
	}
}

func TestResolveScanTimeout_RejectsZero(t *testing.T) {
	t.Setenv("SCAN_TIMEOUT_SECONDS", "0")
	if got := resolveScanTimeout(); got != defaultScanTimeout {
		t.Errorf("zero override should not produce a 0-timeout context; got %v", got)
	}
}

func TestResolveScanTimeout_RejectsNegative(t *testing.T) {
	t.Setenv("SCAN_TIMEOUT_SECONDS", "-5")
	if got := resolveScanTimeout(); got != defaultScanTimeout {
		t.Errorf("negative: got %v, want default", got)
	}
}

func TestDefaultScanTimeout_ValueLock(t *testing.T) {
	// Docker-compose ships SCAN_TIMEOUT_SECONDS=1800 (30m) which matches this.
	if defaultScanTimeout != 30*time.Minute {
		t.Errorf("defaultScanTimeout changed: got %v, want 30m", defaultScanTimeout)
	}
}
