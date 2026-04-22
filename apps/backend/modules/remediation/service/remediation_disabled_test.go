package service

import (
	"context"
	"errors"
	"testing"
)

// These tests verify the REMEDIATION_ENABLED safety gate that blocks both
// single-finding and batch remediation paths when the env flag is not "true".
// The gate exists because the current connector set is stubs — letting
// remediation "succeed" against them would silently report compliance wins
// that didn't actually mutate source data.

func TestIsRemediationEnabled(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want bool
	}{
		{"unset", "", false},
		{"explicit false", "false", false},
		{"lowercase true", "true", true},
		{"uppercase TRUE", "TRUE", true},
		{"mixed case True", "True", true},
		{"yes is not true", "yes", false},
		{"1 is not true", "1", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("REMEDIATION_ENABLED", tc.env)
			if got := isRemediationEnabled(); got != tc.want {
				t.Errorf("REMEDIATION_ENABLED=%q: got %v, want %v", tc.env, got, tc.want)
			}
		})
	}
}

func TestExecuteRemediation_BlockedWhenDisabled(t *testing.T) {
	t.Setenv("REMEDIATION_ENABLED", "false")

	// Safety: the DB must never be touched when the gate is closed.
	// We construct a service with nil db — any attempt to use it panics.
	svc := &RemediationService{}

	_, err := svc.ExecuteRemediation(context.Background(), "finding-id", "MASK", "user-id")
	if err == nil {
		t.Fatal("expected ErrRemediationDisabled")
	}
	if !errors.Is(err, ErrRemediationDisabled) {
		t.Errorf("expected ErrRemediationDisabled, got %v", err)
	}
}

func TestExecuteRemediationRequest_BlockedWhenDisabled(t *testing.T) {
	t.Setenv("REMEDIATION_ENABLED", "false")

	svc := &RemediationService{}
	_, err := svc.ExecuteRemediationRequest(context.Background(), "request-id", "user-id")
	if err == nil {
		t.Fatal("expected ErrRemediationDisabled")
	}
	if !errors.Is(err, ErrRemediationDisabled) {
		t.Errorf("expected ErrRemediationDisabled, got %v", err)
	}
}

func TestExecuteRemediation_GatePrecedesAnyIO(t *testing.T) {
	// When disabled, the gate must short-circuit before any DB/connector IO.
	// We confirm this by passing a nil DB — a real DB call would segfault.
	t.Setenv("REMEDIATION_ENABLED", "false")
	svc := &RemediationService{}

	// Call succeeds (returns err, no panic).
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("gate allowed IO when disabled: panic=%v", r)
		}
	}()
	_, _ = svc.ExecuteRemediation(context.Background(), "id", "MASK", "user")
	_, _ = svc.ExecuteRemediationRequest(context.Background(), "id", "user")
}
