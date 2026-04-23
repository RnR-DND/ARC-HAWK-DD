package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// helper: create db + mock with unordered matching so the async
// maybeAdjustPatternThreshold goroutine can consume its expectation at any point.
func newFeedbackMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	mock.MatchExpectationsInOrder(false)
	t.Cleanup(func() { db.Close() })
	return db, mock
}

func TestRecordCorrection_CorrectionTypes(t *testing.T) {
	cases := []struct {
		name           string
		correctionType string
		userID         string
		wantNilUser    bool
	}{
		{"false_positive", "false_positive", "22222222-2222-2222-2222-222222222222", false},
		{"false_negative", "false_negative", "33333333-3333-3333-3333-333333333333", false},
		{"confirmed", "confirmed", "44444444-4444-4444-4444-444444444444", false},
		{"empty_user_id_gives_nil_correctedBy", "confirmed", "", true},
		{"zero_uuid_gives_nil_correctedBy", "confirmed", "00000000-0000-0000-0000-000000000000", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newFeedbackMock(t)
			const tenantID = "tenant-aaa"
			const findingID = "finding-bbb"
			const patternCode = "IN_PAN"

			// 1. Finding lookup → returns pattern code
			mock.ExpectQuery(`SELECT COALESCE`).
				WithArgs(findingID, tenantID).
				WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(patternCode))

			// 2. Insert correction (corrected_by is nil when user is empty/zero)
			if tc.wantNilUser {
				mock.ExpectExec(`INSERT INTO feedback_corrections`).
					WithArgs(tenantID, findingID, patternCode, tc.correctionType, nil).
					WillReturnResult(sqlmock.NewResult(1, 1))
			} else {
				mock.ExpectExec(`INSERT INTO feedback_corrections`).
					WithArgs(tenantID, findingID, patternCode, tc.correctionType, tc.userID).
					WillReturnResult(sqlmock.NewResult(1, 1))
			}

			// 3. Background goroutine: maybeAdjustPatternThreshold (returns early, count < 10)
			mock.ExpectQuery(`COUNT`).
				WithArgs(tenantID, patternCode).
				WillReturnRows(sqlmock.NewRows([]string{"cnt", "prec"}).AddRow(0, 1.0))

			svc := NewFeedbackService(db)
			err := svc.RecordCorrection(context.Background(), tenantID, tc.userID, findingID, tc.correctionType)
			if err != nil {
				t.Fatalf("RecordCorrection: %v", err)
			}

			time.Sleep(60 * time.Millisecond) // wait for background goroutine

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestRecordCorrection_FindingNotFound(t *testing.T) {
	db, mock := newFeedbackMock(t)

	mock.ExpectQuery(`SELECT COALESCE`).
		WithArgs("missing", "tenant-x").
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"})) // no rows → sql.ErrNoRows

	svc := NewFeedbackService(db)
	err := svc.RecordCorrection(context.Background(), "tenant-x", "user-1", "missing", "confirmed")
	if err == nil {
		t.Error("expected error for missing finding, got nil")
	}
}

func TestRecordCorrection_InsertError(t *testing.T) {
	db, mock := newFeedbackMock(t)
	const tenantID = "tenant-y"
	const findingID = "finding-y"

	mock.ExpectQuery(`SELECT COALESCE`).
		WithArgs(findingID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow("IN_AADHAAR"))

	mock.ExpectExec(`INSERT INTO feedback_corrections`).
		WillReturnError(sql.ErrConnDone)

	svc := NewFeedbackService(db)
	err := svc.RecordCorrection(context.Background(), tenantID, "user-1", findingID, "false_positive")
	if err == nil {
		t.Error("expected insert error, got nil")
	}
}

func TestRecordCorrection_ThresholdAdjustedWhenEnoughData(t *testing.T) {
	db, mock := newFeedbackMock(t)
	const tenantID = "tenant-z"
	const findingID = "finding-z"
	const patternCode = "IN_PAN"

	mock.ExpectQuery(`SELECT COALESCE`).
		WithArgs(findingID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(patternCode))

	mock.ExpectExec(`INSERT INTO feedback_corrections`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Goroutine: ≥10 feedback entries with 80% precision → triggers threshold update
	mock.ExpectQuery(`COUNT`).
		WithArgs(tenantID, patternCode).
		WillReturnRows(sqlmock.NewRows([]string{"cnt", "prec"}).AddRow(15, 0.8))

	mock.ExpectExec(`pattern_confidence_overrides`).
		WithArgs(tenantID, patternCode, 80).
		WillReturnResult(sqlmock.NewResult(1, 1))

	svc := NewFeedbackService(db)
	err := svc.RecordCorrection(context.Background(), tenantID, "user-1", findingID, "confirmed")
	if err != nil {
		t.Fatalf("RecordCorrection: %v", err)
	}

	time.Sleep(80 * time.Millisecond)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
