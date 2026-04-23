package service

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/arc-platform/backend/modules/shared/infrastructure/audit"
	"github.com/google/uuid"
)

var testTenantID = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

// buildDetector returns a detector + two sqlmocks (main DB + audit ledger DB).
func buildDetector(t *testing.T) (*ObligationRegressionDetector, sqlmock.Sqlmock, sqlmock.Sqlmock) {
	t.Helper()

	db, mainMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("main sqlmock: %v", err)
	}
	mainMock.MatchExpectationsInOrder(false)

	auditDB, auditMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("audit sqlmock: %v", err)
	}

	t.Cleanup(func() { db.Close(); auditDB.Close() })
	return NewObligationRegressionDetector(db, audit.NewLedgerLogger(auditDB)), mainMock, auditMock
}

func TestDetectRegressions_NewPIICategory(t *testing.T) {
	detector, mainMock, auditMock := buildDetector(t)
	scanID := "scan-001"

	// current scan has IN_AADHAAR
	mainMock.ExpectQuery(`SELECT DISTINCT pii_category FROM findings`).
		WithArgs(scanID).
		WillReturnRows(sqlmock.NewRows([]string{"pii_category"}).AddRow("IN_AADHAAR"))

	// previous scans have no categories
	mainMock.ExpectQuery(`SELECT DISTINCT f\.pii_category`).
		WithArgs(testTenantID, scanID).
		WillReturnRows(sqlmock.NewRows([]string{"pii_category"}))

	// audit ledger INSERT (for new category)
	auditMock.ExpectExec(`INSERT INTO audit_ledger`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// regression upsert
	mainMock.ExpectExec(`INSERT INTO obligation_regressions`).
		WithArgs(testTenantID, scanID, "IN_AADHAAR").
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := detector.DetectRegressions(context.Background(), testTenantID, scanID); err != nil {
		t.Fatalf("DetectRegressions: %v", err)
	}

	if err := mainMock.ExpectationsWereMet(); err != nil {
		t.Errorf("main DB expectations: %v", err)
	}
	if err := auditMock.ExpectationsWereMet(); err != nil {
		t.Errorf("audit DB expectations: %v", err)
	}
}

func TestDetectRegressions_KnownCategory_NoRegression(t *testing.T) {
	detector, mainMock, auditMock := buildDetector(t)
	scanID := "scan-002"

	// current scan has IN_PAN
	mainMock.ExpectQuery(`SELECT DISTINCT pii_category FROM findings`).
		WithArgs(scanID).
		WillReturnRows(sqlmock.NewRows([]string{"pii_category"}).AddRow("IN_PAN"))

	// previous scans already have IN_PAN
	mainMock.ExpectQuery(`SELECT DISTINCT f\.pii_category`).
		WithArgs(testTenantID, scanID).
		WillReturnRows(sqlmock.NewRows([]string{"pii_category"}).AddRow("IN_PAN"))

	// no audit insert, no regression upsert expected

	if err := detector.DetectRegressions(context.Background(), testTenantID, scanID); err != nil {
		t.Fatalf("DetectRegressions: %v", err)
	}

	if err := mainMock.ExpectationsWereMet(); err != nil {
		t.Errorf("main DB expectations: %v", err)
	}
	if err := auditMock.ExpectationsWereMet(); err != nil {
		t.Errorf("audit DB expectations (should be empty): %v", err)
	}
}

func TestDetectRegressions_MultipleNewCategories(t *testing.T) {
	detector, mainMock, auditMock := buildDetector(t)
	scanID := "scan-003"

	// current: IN_AADHAAR + IN_PAN + HEALTH_RECORD
	mainMock.ExpectQuery(`SELECT DISTINCT pii_category FROM findings`).
		WithArgs(scanID).
		WillReturnRows(sqlmock.NewRows([]string{"pii_category"}).
			AddRow("IN_AADHAAR").
			AddRow("IN_PAN").
			AddRow("HEALTH_RECORD"))

	// previous: only IN_PAN was known
	mainMock.ExpectQuery(`SELECT DISTINCT f\.pii_category`).
		WithArgs(testTenantID, scanID).
		WillReturnRows(sqlmock.NewRows([]string{"pii_category"}).AddRow("IN_PAN"))

	// 2 new categories → 2 audit inserts, 2 regression upserts
	auditMock.ExpectExec(`INSERT INTO audit_ledger`).WillReturnResult(sqlmock.NewResult(1, 1))
	auditMock.ExpectExec(`INSERT INTO audit_ledger`).WillReturnResult(sqlmock.NewResult(1, 1))

	mainMock.ExpectExec(`INSERT INTO obligation_regressions`).WillReturnResult(sqlmock.NewResult(1, 1))
	mainMock.ExpectExec(`INSERT INTO obligation_regressions`).WillReturnResult(sqlmock.NewResult(1, 1))

	if err := detector.DetectRegressions(context.Background(), testTenantID, scanID); err != nil {
		t.Fatalf("DetectRegressions: %v", err)
	}

	if err := mainMock.ExpectationsWereMet(); err != nil {
		t.Errorf("main DB expectations: %v", err)
	}
	if err := auditMock.ExpectationsWereMet(); err != nil {
		t.Errorf("audit DB expectations: %v", err)
	}
}

func TestDetectRegressions_EmptyScan_NoRegressions(t *testing.T) {
	detector, mainMock, _ := buildDetector(t)
	scanID := "scan-004"

	// current scan: no PII found
	mainMock.ExpectQuery(`SELECT DISTINCT pii_category FROM findings`).
		WithArgs(scanID).
		WillReturnRows(sqlmock.NewRows([]string{"pii_category"}))

	// previous categories don't matter — function loops over current categories
	mainMock.ExpectQuery(`SELECT DISTINCT f\.pii_category`).
		WithArgs(testTenantID, scanID).
		WillReturnRows(sqlmock.NewRows([]string{"pii_category"}).AddRow("IN_PAN"))

	if err := detector.DetectRegressions(context.Background(), testTenantID, scanID); err != nil {
		t.Fatalf("DetectRegressions: %v", err)
	}
}
