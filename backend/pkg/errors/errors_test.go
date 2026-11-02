package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppErrorFormatting(t *testing.T) {
	appErr := InvalidInput("source_code is required")
	if appErr.Code != "INVALID_INPUT" || appErr.HTTPStatus != http.StatusBadRequest {
		t.Errorf("error = %+v", appErr)
	}
	if appErr.Error() != "INVALID_INPUT: source_code is required" {
		t.Errorf("Error() = %q", appErr.Error())
	}
}

func TestAsUsesWrappedAppError(t *testing.T) {
	cause := errors.New("database is down")
	wrapped := Wrap(cause, "SERVICE_UNAVAILABLE", "audit service is unavailable", http.StatusServiceUnavailable)
	wrappedAgain := errors.Join(errors.New("request failed"), wrapped)

	resolved := As(wrappedAgain)
	if resolved.Code != "SERVICE_UNAVAILABLE" || !errors.Is(resolved, cause) {
		t.Errorf("resolved = %+v", resolved)
	}
}

func TestAsDefaultsToInternal(t *testing.T) {
	resolved := As(errors.New("boom"))
	if resolved.Code != "INTERNAL_ERROR" || resolved.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("resolved = %+v", resolved)
	}
}

func TestErrorConstructors(t *testing.T) {
	cases := []struct {
		err    *AppError
		code   string
		status int
	}{
		{ParseError("bad"), "PARSE_ERROR", http.StatusUnprocessableEntity},
		{AuditNotFound(), "AUDIT_NOT_FOUND", http.StatusNotFound},
		{AuditNotReady(), "AUDIT_NOT_READY", http.StatusConflict},
		{ServiceUnavailable("down"), "SERVICE_UNAVAILABLE", http.StatusServiceUnavailable},
	}

	for _, testCase := range cases {
		if testCase.err.Code != testCase.code || testCase.err.HTTPStatus != testCase.status {
			t.Errorf("%s error = %+v", testCase.code, testCase.err)
		}
	}
}
