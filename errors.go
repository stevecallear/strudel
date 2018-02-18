package strudel

import "strings"

type (
	// Fields represents a set of error fields
	Fields map[string]interface{}

	// Error represents an error
	Error struct {
		msg       string
		code      int
		fields    Fields
		logFields Fields
	}
)

// NewError returns a new error
func NewError(msg string) *Error {
	return &Error{
		msg:       msg,
		fields:    Fields{},
		logFields: Fields{},
	}
}

// WithCode sets the error code
func (e *Error) WithCode(code int) *Error {
	e.code = code
	return e
}

// WithField adds the specified error field
func (e *Error) WithField(key string, value interface{}) *Error {
	if strings.TrimSpace(key) != "" {
		e.fields[key] = value
	}
	return e
}

// WithFields adds the specified error fields
func (e *Error) WithFields(f Fields) *Error {
	for k, v := range f {
		e.WithField(k, v)
	}
	return e
}

// WithLogField adds the specified log-only error field
func (e *Error) WithLogField(key string, value interface{}) *Error {
	if strings.TrimSpace(key) != "" {
		e.logFields[key] = value
	}
	return e
}

// WithLogFields adds the specified log-only error fields
func (e *Error) WithLogFields(f Fields) *Error {
	for k, v := range f {
		e.WithLogField(k, v)
	}
	return e
}

// Error returns the error message
func (e *Error) Error() string {
	return e.msg
}

// Code returns the error code
func (e *Error) Code() int {
	return e.code
}

// Fields returns all error fields that are not log-only
func (e *Error) Fields() Fields {
	return e.fields
}

// LogFields returns all error fields including those that are log-only
func (e *Error) LogFields() Fields {
	f := make(Fields, len(e.fields)+len(e.logFields))
	for k, v := range e.fields {
		f[k] = v
	}
	for k, v := range e.logFields {
		f[k] = v
	}
	return f
}
