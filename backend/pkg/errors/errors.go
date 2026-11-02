// Package errors defines the structured application error used by every API
// endpoint. Each error carries a stable machine-readable code and the HTTP
// status it should be rendered with.
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError is a domain error that maps onto a structured JSON API response.
type AppError struct {
	Code       string
	Message    string
	HTTPStatus int
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New builds an AppError.
func New(code, message string, status int) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: status}
}

// Wrap builds an AppError with an underlying cause that is logged but never
// exposed to API clients.
func Wrap(err error, code, message string, status int) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: status, Err: err}
}

func InvalidInput(message string) *AppError {
	return New("INVALID_INPUT", message, http.StatusBadRequest)
}

func ParseError(message string) *AppError {
	return New("PARSE_ERROR", message, http.StatusUnprocessableEntity)
}

func AuditNotFound() *AppError {
	return New("AUDIT_NOT_FOUND", "audit not found", http.StatusNotFound)
}

func AuditNotReady() *AppError {
	return New("AUDIT_NOT_READY", "audit report is not ready yet", http.StatusConflict)
}

func UnsupportedMediaType(message string) *AppError {
	return New("UNSUPPORTED_MEDIA_TYPE", message, http.StatusUnsupportedMediaType)
}

func ServiceUnavailable(message string) *AppError {
	return New("SERVICE_UNAVAILABLE", message, http.StatusServiceUnavailable)
}

func Internal(err error) *AppError {
	return Wrap(err, "INTERNAL_ERROR", "internal server error", http.StatusInternalServerError)
}

// As converts any error into an *AppError, defaulting to an internal error.
func As(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal(err)
}
