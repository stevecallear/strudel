package strudel_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/stevecallear/strudel"
)

func TestNewError(t *testing.T) {
	t.Run("should set the message", func(t *testing.T) {
		const exp = "error"
		act := strudel.NewError(exp).Error()
		if act != exp {
			t.Errorf("got %s, expected %s", act, exp)
		}
	})
}

func TestError_WithCode(t *testing.T) {
	tests := []struct {
		name string
		err  *strudel.Error
		code int
		exp  int
	}{
		{
			name: "should set the code",
			err:  strudel.NewError("error"),
			code: http.StatusNotFound,
			exp:  http.StatusNotFound,
		},
		{
			name: "should overwrite the existing code",
			err:  strudel.NewError("error").WithCode(http.StatusConflict),
			code: http.StatusNotFound,
			exp:  http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		act := tt.err.WithCode(tt.code).Code()
		if act != tt.exp {
			t.Errorf("got %d, expected %d", act, tt.exp)
		}
	}
}

func TestError_WithField(t *testing.T) {
	tests := []struct {
		name  string
		err   *strudel.Error
		key   string
		value interface{}
		exp   strudel.Fields
	}{
		{
			name:  "should ignore empty key",
			err:   strudel.NewError("error"),
			key:   " \n\t",
			value: "value",
			exp:   strudel.Fields{},
		},
		{
			name:  "should set the value",
			err:   strudel.NewError("error"),
			key:   "key",
			value: "value",
			exp:   strudel.Fields{"key": "value"},
		},
		{
			name:  "should preserve existing values",
			err:   strudel.NewError("error").WithField("keyA", "valueA"),
			key:   "keyB",
			value: "valueB",
			exp:   strudel.Fields{"keyA": "valueA", "keyB": "valueB"},
		},
		{
			name:  "should overwrite existing keys",
			err:   strudel.NewError("error").WithField("key", "valueA"),
			key:   "key",
			value: "valueB",
			exp:   strudel.Fields{"key": "valueB"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			act := tt.err.WithField(tt.key, tt.value).Fields()
			if !reflect.DeepEqual(act, tt.exp) {
				t.Errorf("got %v, expected %v", act, tt.exp)
			}
		})
	}
}

func TestError_WithFields(t *testing.T) {
	val := &struct{}{}
	tests := []struct {
		name   string
		err    *strudel.Error
		fields strudel.Fields
		exp    strudel.Fields
	}{
		{
			name:   "should ignore empty keys",
			err:    strudel.NewError("error"),
			fields: strudel.Fields{" \n\t": "valueA", "key": "valueB"},
			exp:    strudel.Fields{"key": "valueB"},
		},
		{
			name:   "should set the value",
			err:    strudel.NewError("error"),
			fields: strudel.Fields{"key": "value"},
			exp:    strudel.Fields{"key": "value"},
		},
		{
			name:   "should preserve existing values",
			err:    strudel.NewError("error").WithField("keyA", "valueA"),
			fields: strudel.Fields{"keyB": "valueB"},
			exp:    strudel.Fields{"keyA": "valueA", "keyB": "valueB"},
		},
		{
			name:   "should overwrite existing keys",
			err:    strudel.NewError("error").WithField("key", "valueA"),
			fields: strudel.Fields{"key": "valueB", "keyC": "valueC"},
			exp:    strudel.Fields{"key": "valueB", "keyC": "valueC"},
		},
		{
			name:   "should shallow copy fields",
			err:    strudel.NewError("error"),
			fields: strudel.Fields{"key": val},
			exp:    strudel.Fields{"key": val},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			act := tt.err.WithFields(tt.fields).Fields()
			if !reflect.DeepEqual(act, tt.exp) {
				t.Errorf("got %v, expected %v", act, tt.exp)
			}
		})
	}
}
