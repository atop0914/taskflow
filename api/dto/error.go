package dto

import "fmt"

// ErrorCode 错误码定义
type ErrorCode int

const (
	// Success codes
	CodeSuccess ErrorCode = 0

	// Client errors (4xx)
	CodeBadRequest     ErrorCode = 400
	CodeUnauthorized   ErrorCode = 401
	CodeForbidden       ErrorCode = 403
	CodeNotFound        ErrorCode = 404
	CodeTooManyRequests ErrorCode = 429

	// Server errors (5xx)
	CodeInternalError   ErrorCode = 500
	CodeServiceUnavailable ErrorCode = 503

	// Business errors (6xxx)
	CodeTooManyNames    ErrorCode = 6001
	CodeStatsDisabled   ErrorCode = 6002
	CodeValidationError ErrorCode = 6003
)

// Error 业务错误
type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("code: %d, message: %s", e.Code, e.Message)
}

// NewBusinessError 创建业务错误
func NewBusinessError(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// NewValidationError 创建验证错误
func NewValidationError(message string, details string) *Error {
	return &Error{
		Code:    CodeValidationError,
		Message: message,
		Details: details,
	}
}

// Predefined errors
var (
	ErrBadRequest = &Error{Code: CodeBadRequest, Message: "bad request"}
	ErrNotFound = &Error{Code: CodeNotFound, Message: "resource not found"}
	ErrTooManyNames = &Error{Code: CodeTooManyNames, Message: "too many names, maximum allowed"}
	ErrStatsDisabled = &Error{Code: CodeStatsDisabled, Message: "statistics feature is disabled"}
)
