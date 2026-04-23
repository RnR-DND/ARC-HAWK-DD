package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func newWatchdogMock(t *testing.T) (*ScanWatchdog, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewScanWatchdog(db, 5*time.Minute), mock
}

func TestScanWatchdog_ReapStalled_NoStalls(t *testing.T) {
	w, mock := newWatchdogMock(t)

	mock.ExpectExec(`UPDATE scan_runs`).
		WithArgs(sqlmock.AnyArg()). // cutoff timestamp
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := w.reapStalled(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestScanWatchdog_ReapStalled_TwoStalls(t *testing.T) {
	w, mock := newWatchdogMock(t)

	mock.ExpectExec(`UPDATE scan_runs`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(2, 2))

	if err := w.reapStalled(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestScanWatchdog_ReapStalled_DBError(t *testing.T) {
	w, mock := newWatchdogMock(t)

	mock.ExpectExec(`UPDATE scan_runs`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrConnDone)

	if err := w.reapStalled(context.Background()); err == nil {
		t.Error("expected error from DB, got nil")
	}
}

func TestNewScanWatchdog_DefaultInterval(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	// Passing interval ≤ 0 should default to 5 minutes
	w := NewScanWatchdog(db, 0)
	if w.interval != 5*time.Minute {
		t.Errorf("default interval = %v, want 5m", w.interval)
	}
}

func TestScanWatchdog_StaleThreshold(t *testing.T) {
	// stalledScanThreshold is 2 hours; the cutoff must be at least 2h in the past
	if stalledScanThreshold != 2*time.Hour {
		t.Errorf("stalledScanThreshold = %v, want 2h", stalledScanThreshold)
	}
}
