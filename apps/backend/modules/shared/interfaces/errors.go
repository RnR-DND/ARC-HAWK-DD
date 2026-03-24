package interfaces

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Error codes
const (
	ErrCodeInternalServer   = "INTERNAL_SERVER_ERROR"
	ErrCodeBadRequest       = "BAD_REQUEST"
	ErrCodeUnauthorized     = "UNAUTHORIZED"
	ErrCodeForbidden        = "FORBIDDEN"
	ErrCodeNotFound         = "NOT_FOUND"
	ErrCodeValidationFailed = "VALIDATION_FAILED"
	ErrCodeConflict         = "CONFLICT"
)

// NewErrorResponse creates a new error response
func NewErrorResponse(code, message string, details interface{}) ErrorResponse {
	return ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	}
}
