package strudel

import "strings"

type (
	// Fields represents a set of error fields
	Fields map[string]interface{}

	// Error represents an error
	Error struct {
		msg    string
		code   int
		fields Fields
	}
)

// NewError returns a new error
func NewError(msg string) *Error {
	return &Error{
		msg:    msg,
		fields: Fields{},
	}
}

// WithCode sets the specified error code
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

// Error returns the error message
func (e *Error) Error() string {
	return e.msg
}

// Code returns the error code
func (e *Error) Code() int {
	return e.code
}

// Fields returns the error fields
func (e *Error) Fields() Fields {
	return e.fields
}
