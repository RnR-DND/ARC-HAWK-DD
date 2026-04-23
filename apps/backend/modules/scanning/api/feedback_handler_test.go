package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	authmiddleware "github.com/arc-platform/backend/modules/auth/middleware"
	"github.com/arc-platform/backend/modules/scanning/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	feedbackTenantID  = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	feedbackUserID    = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	feedbackFindingID = "33333333-3333-3333-3333-333333333333"
)

func newFeedbackHandlerRouter(t *testing.T) (*gin.Engine, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	mock.MatchExpectationsInOrder(false)
	t.Cleanup(func() { db.Close() })

	svc := service.NewFeedbackService(db)
	h := NewFeedbackHandler(svc)

	r := gin.New()
	injectCtx := func(c *gin.Context) {
		c.Set("tenant_context", authmiddleware.TenantContext{
			TenantID: feedbackTenantID,
			UserID:   feedbackUserID,
			Role:     "admin",
		})
		c.Next()
	}
	r.POST("/findings/:id/feedback", injectCtx, h.SubmitFeedback)
	r.GET("/patterns/precision", injectCtx, h.GetPatternPrecision)
	return r, mock
}

func TestSubmitFeedback_HappyPath_FalsePositive(t *testing.T) {
	r, mock := newFeedbackHandlerRouter(t)

	mock.ExpectQuery(`SELECT COALESCE`).
		WithArgs(feedbackFindingID, feedbackTenantID.String()).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow("IN_PAN"))

	mock.ExpectExec(`INSERT INTO feedback_corrections`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// goroutine: threshold check (count < 10)
	mock.ExpectQuery(`COUNT`).
		WillReturnRows(sqlmock.NewRows([]string{"cnt", "prec"}).AddRow(0, 1.0))

	body, _ := json.Marshal(map[string]string{"correction_type": "false_positive"})
	req := httptest.NewRequest(http.MethodPost, "/findings/"+feedbackFindingID+"/feedback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	time.Sleep(60 * time.Millisecond)
}

func TestSubmitFeedback_HappyPath_Confirmed(t *testing.T) {
	r, mock := newFeedbackHandlerRouter(t)

	mock.ExpectQuery(`SELECT COALESCE`).
		WithArgs(feedbackFindingID, feedbackTenantID.String()).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow("IN_AADHAAR"))

	mock.ExpectExec(`INSERT INTO feedback_corrections`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(`COUNT`).
		WillReturnRows(sqlmock.NewRows([]string{"cnt", "prec"}).AddRow(0, 1.0))

	body, _ := json.Marshal(map[string]string{"correction_type": "confirmed"})
	req := httptest.NewRequest(http.MethodPost, "/findings/"+feedbackFindingID+"/feedback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	time.Sleep(60 * time.Millisecond)
}

func TestSubmitFeedback_MissingBody_Returns400(t *testing.T) {
	r, _ := newFeedbackHandlerRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/findings/"+feedbackFindingID+"/feedback", nil)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSubmitFeedback_InvalidCorrectionType_Returns400(t *testing.T) {
	r, _ := newFeedbackHandlerRouter(t)

	body, _ := json.Marshal(map[string]string{"correction_type": "bogus_type"})
	req := httptest.NewRequest(http.MethodPost, "/findings/"+feedbackFindingID+"/feedback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid correction_type, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSubmitFeedback_FindingNotFound_Returns500(t *testing.T) {
	r, mock := newFeedbackHandlerRouter(t)

	// DB returns no rows for the finding
	mock.ExpectQuery(`SELECT COALESCE`).
		WithArgs(feedbackFindingID, feedbackTenantID.String()).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}))

	body, _ := json.Marshal(map[string]string{"correction_type": "confirmed"})
	req := httptest.NewRequest(http.MethodPost, "/findings/"+feedbackFindingID+"/feedback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestGetPatternPrecision_Returns200(t *testing.T) {
	r, mock := newFeedbackHandlerRouter(t)

	mock.ExpectQuery(`SELECT`).
		WithArgs(feedbackTenantID.String()).
		WillReturnRows(sqlmock.NewRows([]string{"pattern_code", "confirmed", "false_positives", "precision"}).
			AddRow("IN_PAN", 5, 2, 0.71))

	req := httptest.NewRequest(http.MethodGet, "/patterns/precision", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
