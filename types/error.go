package types

import (
	"encoding/json"
	"fmt"
)

// ErrorResponse represents http APIs errors
type ErrorResponse struct {
	Errors []RestError `json:"errors"`
}

func (e *ErrorResponse) Error() string {
	b, _ := json.Marshal(e)
	return string(b)
}

// RestError represents http APIs error
type RestError struct {
	ErrorCode   int    `json:"errorCode"`
	ErrorReason string `json:"errorReason"`
	ErrorSource string `json:"errorSource"`
}

func (e RestError) Error() string {
	return fmt.Sprintf("errorCode=%d | errorReason=%s | errorSource=%s", e.ErrorCode, e.ErrorSource, e.ErrorReason)
}
