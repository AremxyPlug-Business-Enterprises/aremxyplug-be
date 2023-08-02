package errors

// WithStatus Terror optional attribute
func WithStatus(status int) TerrorOptionalAttrs {
	return func(t *Terror) {
		t.status = status
	}
}

// WithInstance Terror optional attribute
func WithInstance(instance string) TerrorOptionalAttrs {
	return func(t *Terror) {
		t.instance = instance
	}
}

// WithTraceID Terror optional attribute
func WithTraceID(traceID string) TerrorOptionalAttrs {
	return func(t *Terror) {
		t.traceID = traceID
	}
}

// WithHelp Terror optional attribute
func WithHelp(help string) TerrorOptionalAttrs {
	return func(t *Terror) {
		t.help = help
	}
}
