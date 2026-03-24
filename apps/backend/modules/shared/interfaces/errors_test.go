package interfaces

import (
	"testing"
)

func TestNewErrorResponse(t *testing.T) {
	errResp := NewErrorResponse("TEST_ERROR", "Test message", "details")

	if errResp.Code != "TEST_ERROR" {
		t.Errorf("Expected code TEST_ERROR, got %s", errResp.Code)
	}
	if errResp.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got %s", errResp.Message)
	}
	if errResp.Details != "details" {
		t.Errorf("Expected details 'details', got %v", errResp.Details)
	}
}
