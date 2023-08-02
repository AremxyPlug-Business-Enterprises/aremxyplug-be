package errors

import "encoding/json"

// Ensure Terror implements error interface
var _ error = &Terror{}

// Terror defines a aremxy error structure
type Terror struct {
	code      int
	errorType string
	message   string
	status    int
	detail    string
	traceID   string
	instance  string
	help      string
}

// Code getter
func (t *Terror) Code() int {
	return t.code
}

// ErrorType getter
func (t *Terror) ErrorType() string {
	return t.errorType
}

// Message getter
func (t *Terror) Message() string {
	return t.message
}

// Status getter
func (t *Terror) Status() int {
	return t.status
}

// Detail getter
func (t *Terror) Detail() string {
	return t.detail
}

// TraceID getter
func (t *Terror) TraceID() string {
	return t.traceID
}

// Instance getter
func (t *Terror) Instance() string {
	return t.instance
}

// Help getter
func (t *Terror) Help() string {
	return t.help
}

// terrorJSONModel represents the json sharable model of a Roava Terror
// for internal use only
type terrorJSONModel struct {
	terrorError `json:"error"`
}

type terrorError struct {
	Code      int    `json:"code"`
	ErrorType string `json:"type"`
	Message   string `json:"message"`
	Status    int    `json:"status,omitempty"`
	Detail    string `json:"detail"`
	TraceID   string `json:"trace_id,omitempty"`
	Instance  string `json:"instance,omitempty"`
	Help      string `json:"help,omitempty"`
}

// Error returns a json string representation of the Roava Terror
func (t *Terror) Error() string {
	// Get model json string
	jsonBytes, _ := json.Marshal(terrorJSONModel{
		terrorError{
			Code:      t.code,
			ErrorType: t.errorType,
			Message:   t.message,
			Status:    t.status,
			Detail:    t.detail,
			TraceID:   t.traceID,
			Instance:  t.instance,
			Help:      t.help,
		},
	})

	return string(jsonBytes)
}

// TerrorOptionalAttrs type
type TerrorOptionalAttrs func(t *Terror)

// NewTerror returns an instance of a Terror with the given attributes
func NewTerror(
	code int,
	errorType string,
	message string,
	detail string,
	optionalAttrs ...TerrorOptionalAttrs,
) *Terror {
	terror := &Terror{
		code:      code,
		errorType: errorType,
		message:   message,
		detail:    detail,
	}
	// Execute additional attributes
	for _, optionalAttr := range optionalAttrs {
		optionalAttr(terror)
	}

	return terror
}

// NewTerrorFromJSONString returns an instance of a Terror from the given json string
func NewTerrorFromJSONString(jsonString string) (*Terror, error) {
	var terrorJSON terrorJSONModel
	err := json.Unmarshal([]byte(jsonString), &terrorJSON)
	if err != nil {
		return nil, err
	}

	return &Terror{
		code:      terrorJSON.Code,
		errorType: terrorJSON.ErrorType,
		message:   terrorJSON.Message,
		status:    terrorJSON.Status,
		detail:    terrorJSON.Detail,
		traceID:   terrorJSON.TraceID,
		instance:  terrorJSON.Instance,
		help:      terrorJSON.Help,
	}, nil
}
