package service

import "testing"

// These tests lock in the "partial" terminal state added by the system audit
// (P0-5 / ingest-retry work). If someone reverts the partial transition or
// forgets to add partial to the terminal-state bucket, CI catches it.

func TestValidateStatusTransition_RunningToPartialAllowed(t *testing.T) {
	if err := ValidateStatusTransition(ScanStatusRunning, ScanStatusPartial); err != nil {
		t.Errorf("running → partial should be allowed: %v", err)
	}
}

func TestValidateStatusTransition_PartialIsTerminal(t *testing.T) {
	// Nothing should be allowed out of partial — it is a terminal state.
	for _, to := range []string{
		ScanStatusRunning,
		ScanStatusCompleted,
		ScanStatusFailed,
		ScanStatusCancelled,
		ScanStatusTimeout,
	} {
		if err := ValidateStatusTransition(ScanStatusPartial, to); err == nil {
			t.Errorf("partial → %s must be rejected (partial is terminal)", to)
		}
	}
}

func TestValidateStatusTransition_PendingToPartialRejected(t *testing.T) {
	// Partial should only be reachable from running (after ingest attempted).
	// Not from pending.
	if err := ValidateStatusTransition(ScanStatusPending, ScanStatusPartial); err == nil {
		t.Error("pending → partial must be rejected")
	}
}

func TestValidateStatusTransition_UnknownStateRejected(t *testing.T) {
	if err := ValidateStatusTransition("bogus", ScanStatusRunning); err == nil {
		t.Error("unknown source state must be rejected")
	}
}

func TestScanStatusPartial_Constant(t *testing.T) {
	// Lock the wire string — the scanner and backend both read this.
	if ScanStatusPartial != "partial" {
		t.Errorf("ScanStatusPartial changed: got %q, want partial", ScanStatusPartial)
	}
}
