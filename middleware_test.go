package strudel_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stevecallear/janice"

	"github.com/stevecallear/strudel"
)

func TestRequestLogging(t *testing.T) {
	err := errors.New("error")
	tests := []struct {
		name string
		req  *http.Request
		fn   janice.HandlerFunc
		exp  map[string]interface{}
		err  error
	}{
		{
			name: "should handle errors",
			req:  httptest.NewRequest("GET", "/path", nil),
			fn: func(w http.ResponseWriter, _ *http.Request) error {
				w.WriteHeader(http.StatusInternalServerError)
				return err
			},
			exp: map[string]interface{}{
				"type":   "request",
				"method": "GET",
				"host":   "example.com",
				"path":   "/path",
				"code":   "500",
			},
			err: err,
		},
		{
			name: "should log the status code",
			req:  httptest.NewRequest("GET", "/path", nil),
			fn: func(w http.ResponseWriter, _ *http.Request) error {
				w.WriteHeader(http.StatusMovedPermanently)
				return nil
			},
			exp: map[string]interface{}{
				"type":   "request",
				"method": "GET",
				"host":   "example.com",
				"path":   "/path",
				"code":   "301",
			},
		},
		{
			name: "should log the method",
			req:  httptest.NewRequest("POST", "/path", nil),
			fn: func(w http.ResponseWriter, _ *http.Request) error {
				w.WriteHeader(http.StatusCreated)
				return nil
			},
			exp: map[string]interface{}{
				"type":   "request",
				"method": "POST",
				"host":   "example.com",
				"path":   "/path",
				"code":   "201",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bytes.NewBuffer(nil)
			logTo(b, func() {
				rec := httptest.NewRecorder()
				err := strudel.RequestLogging(tt.fn)(rec, tt.req)
				if err != tt.err {
					t.Errorf("got %v, expected %v", err, tt.err)
				}
				act := map[string]interface{}{}
				if b.Len() > 0 {
					if err = json.Unmarshal(b.Bytes(), &act); err != nil {
						t.Errorf("got %v, expected nil", err)
					}
				}
				for k, v := range tt.exp {
					if act[k] != v {
						t.Errorf("got %s:%v, expected %s:%v", k, act[k], k, v)
					}
				}
			})
		})
	}
}

func TestRecovery(t *testing.T) {
	err := errors.New("error")
	tests := []struct {
		name string
		fn   func() error
		code int
		exp  map[string]interface{}
		err  error
	}{
		{
			name: "should do nothing if there is no panic",
			fn: func() error {
				return nil
			},
			code: http.StatusOK,
			exp:  map[string]interface{}{},
		},
		{
			name: "should pass through errors",
			fn: func() error {
				return err
			},
			code: http.StatusOK,
			exp:  map[string]interface{}{},
			err:  err,
		},
		{
			name: "should recover from panic and log the message",
			fn: func() error {
				panic(err)
			},
			code: http.StatusInternalServerError,
			exp: map[string]interface{}{
				"type":  "recovery",
				"level": "error",
				"msg":   "error",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bytes.NewBuffer(nil)
			logTo(b, func() {
				rec := httptest.NewRecorder()
				err := strudel.Recovery(func(http.ResponseWriter, *http.Request) error {
					return tt.fn()
				})(rec, nil)

				if err != tt.err {
					t.Errorf("got %v, expected %v", err, tt.err)
				}
				if rec.Code != tt.code {
					t.Errorf("got %d, expected %d", rec.Code, tt.code)
				}
				act := map[string]interface{}{}
				if b.Len() > 0 {
					if err = json.Unmarshal(b.Bytes(), &act); err != nil {
						t.Errorf("got %v, expected nil", err)
					}
					delete(act, "time")
				}
				if !reflect.DeepEqual(act, tt.exp) {
					t.Errorf("got %v, expected %v", act, tt.exp)
				}
			})
		})
	}
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code int
		body map[string]interface{}
		log  map[string]interface{}
	}{
		{
			name: "should do nothing if there is no error",
			code: http.StatusOK,
			body: map[string]interface{}{},
			log:  map[string]interface{}{},
		},
		{
			name: "should not write other errors to the body",
			err:  errors.New("error"),
			code: http.StatusInternalServerError,
			body: map[string]interface{}{
				"status":  "error",
				"message": http.StatusText(http.StatusInternalServerError),
			},
			log: map[string]interface{}{
				"type":  "error",
				"level": "error",
				"msg":   "error",
			},
		},
		{
			name: "should not log error code if not specified",
			err:  strudel.NewError("error"),
			code: http.StatusInternalServerError,
			body: map[string]interface{}{
				"status":  "error",
				"message": "error",
			},
			log: map[string]interface{}{
				"type":  "error",
				"level": "error",
				"msg":   "error",
			},
		},
		{
			name: "should use status 500 if code is not HTTP status code",
			err:  strudel.NewError("error").WithCode(1),
			code: http.StatusInternalServerError,
			body: map[string]interface{}{
				"status":  "error",
				"message": "error",
			},
			log: map[string]interface{}{
				"type":  "error",
				"level": "error",
				"code":  1,
				"msg":   "error",
			},
		},
		{
			name: "should use status 4xx if specified as error code",
			err:  strudel.NewError("error").WithCode(http.StatusNotFound),
			code: http.StatusNotFound,
			body: map[string]interface{}{
				"status":  "fail",
				"message": "error",
				"data":    nil,
			},
			log: map[string]interface{}{
				"type":  "error",
				"level": "error",
				"code":  http.StatusNotFound,
				"msg":   "error",
			},
		},
		{
			name: "should use status 5xx if specified as error code",
			err:  strudel.NewError("error").WithCode(http.StatusServiceUnavailable),
			code: http.StatusServiceUnavailable,
			body: map[string]interface{}{
				"status":  "error",
				"message": "error",
			},
			log: map[string]interface{}{
				"type":  "error",
				"level": "error",
				"code":  http.StatusServiceUnavailable,
				"msg":   "error",
			},
		},
		{
			name: "should write error fields",
			err:  strudel.NewError("error").WithField("key", "value"),
			code: http.StatusInternalServerError,
			body: map[string]interface{}{
				"status":  "error",
				"message": "error",
				"data":    map[string]interface{}{"key": "value"},
			},
			log: map[string]interface{}{
				"type":  "error",
				"level": "error",
				"msg":   "error",
				"data":  map[string]interface{}{"key": "value"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bytes.NewBuffer(nil)
			logTo(b, func() {
				rec := httptest.NewRecorder()
				err := strudel.ErrorHandling(func(http.ResponseWriter, *http.Request) error {
					return tt.err
				})(rec, nil)

				if err != nil {
					t.Errorf("got %v, expected nil", err)
				}
				if rec.Code != tt.code {
					t.Errorf("got %d, expected %d", rec.Code, tt.code)
				}

				body := map[string]interface{}{}
				if rec.Body.Len() > 0 {
					if err = json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
						t.Errorf("got %v, expected nil", err)
					}
				}
				if !reflect.DeepEqual(body, tt.body) {
					t.Errorf("got %v, expected %v", body, tt.body)
				}

				log := map[string]interface{}{}
				if b.Len() > 0 {
					if err = json.Unmarshal(b.Bytes(), &log); err != nil {
						t.Errorf("got %v, expected nil", err)
					}
					delete(log, "time")
					if c, ok := log["code"]; ok {
						log["code"] = int(c.(float64))
					}
				}
				if !reflect.DeepEqual(log, tt.log) {
					t.Errorf("got %#v, expected %#v", log, tt.log)
				}
			})
		})
	}
}

func logTo(b *bytes.Buffer, fn func()) {
	pl := strudel.Logger
	defer func() {
		strudel.Logger = pl
	}()
	l := logrus.New()
	l.Formatter = new(logrus.JSONFormatter)
	l.Out = b
	strudel.Logger = l
	fn()
}
