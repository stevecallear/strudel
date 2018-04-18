package strudel_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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
		name  string
		reqID string
		req   *http.Request
		fn    janice.HandlerFunc
		exp   map[string]interface{}
		err   error
	}{
		{
			name:  "should handle errors",
			reqID: "requestId",
			req:   httptest.NewRequest("GET", "/path", nil),
			fn: func(w http.ResponseWriter, _ *http.Request) error {
				w.WriteHeader(http.StatusInternalServerError)
				return err
			},
			exp: map[string]interface{}{
				"type":    "request",
				"request": "requestId",
				"method":  "GET",
				"host":    "example.com",
				"path":    "/path",
				"code":    "500",
			},
			err: err,
		},
		{
			name:  "should log the status code",
			reqID: "requestId",
			req:   httptest.NewRequest("GET", "/path", nil),
			fn: func(w http.ResponseWriter, _ *http.Request) error {
				w.WriteHeader(http.StatusMovedPermanently)
				return nil
			},
			exp: map[string]interface{}{
				"type":    "request",
				"request": "requestId",
				"method":  "GET",
				"host":    "example.com",
				"path":    "/path",
				"code":    "301",
			},
		},
		{
			name:  "should log the method",
			reqID: "requestId",
			req:   httptest.NewRequest("POST", "/path", nil),
			fn: func(w http.ResponseWriter, _ *http.Request) error {
				w.WriteHeader(http.StatusCreated)
				return nil
			},
			exp: map[string]interface{}{
				"type":    "request",
				"request": "requestId",
				"method":  "POST",
				"host":    "example.com",
				"path":    "/path",
				"code":    "201",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			mw := janice.New(withNextID(tt.reqID), withLogger(buf))
			rec := httptest.NewRecorder()
			err := mw.Append(strudel.RequestLogging)(tt.fn)(rec, tt.req)
			if err != tt.err {
				t.Errorf("got %v, expected %v", err, tt.err)
			}
			act := map[string]interface{}{}
			if buf.Len() > 0 {
				if err = json.Unmarshal(buf.Bytes(), &act); err != nil {
					t.Errorf("got %v, expected nil", err)
				}
			}
			for k, v := range tt.exp {
				if act[k] != v {
					t.Errorf("got %s:%v, expected %s:%v", k, act[k], k, v)
				}
			}
		})
	}
}

func TestRecovery(t *testing.T) {
	err := errors.New("error")
	tests := []struct {
		name  string
		reqID string
		fn    func() error
		code  int
		exp   map[string]interface{}
		err   error
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
		{
			name:  "should log the request id if it has been set",
			reqID: "requestId",
			fn: func() error {
				panic(err)
			},
			code: http.StatusInternalServerError,
			exp: map[string]interface{}{
				"type":    "recovery",
				"level":   "error",
				"request": "requestId",
				"msg":     "error",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			mw := janice.New(withNextID(tt.reqID), withLogger(bytes.NewBuffer(nil)))
			if tt.reqID != "" {
				mw = mw.Append(strudel.RequestLogging)
			}
			rec, req := httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil)
			err := mw.Append(withLogger(buf), strudel.Recovery)(func(http.ResponseWriter, *http.Request) error {
				return tt.fn()
			})(rec, req)

			if err != tt.err {
				t.Errorf("got %v, expected %v", err, tt.err)
			}
			if rec.Code != tt.code {
				t.Errorf("got %d, expected %d", rec.Code, tt.code)
			}
			act := map[string]interface{}{}
			if buf.Len() > 0 {
				if err = json.Unmarshal(buf.Bytes(), &act); err != nil {
					t.Errorf("got %v, expected nil", err)
				}
				delete(act, "time")
			}
			if !reflect.DeepEqual(act, tt.exp) {
				t.Errorf("got %v, expected %v", act, tt.exp)
			}
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
			name: "should write error fields to log and body",
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
		{
			name: "should not write error log fields to body",
			err:  strudel.NewError("error").WithField("field", "value").WithLogField("logField", "value"),
			code: http.StatusInternalServerError,
			body: map[string]interface{}{
				"status":  "error",
				"message": "error",
				"data":    map[string]interface{}{"field": "value"},
			},
			log: map[string]interface{}{
				"type":  "error",
				"level": "error",
				"msg":   "error",
				"data":  map[string]interface{}{"field": "value", "logField": "value"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			mw := janice.New(withLogger(buf))
			rec := httptest.NewRecorder()
			err := mw.Append(strudel.ErrorHandling)(func(http.ResponseWriter, *http.Request) error {
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
			if buf.Len() > 0 {
				if err = json.Unmarshal(buf.Bytes(), &log); err != nil {
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
	}
}

func withLogger(w io.Writer) janice.MiddlewareFunc {
	nl := func() *logrus.Logger {
		l := logrus.New()
		l.Formatter = new(logrus.JSONFormatter)
		l.Out = w
		return l
	}
	return func(n janice.HandlerFunc) janice.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) error {
			pl := strudel.Logger
			defer func() {
				strudel.Logger = pl
			}()
			strudel.Logger = nl()
			return n(w, r)
		}
	}
}

func withNextID(v string) janice.MiddlewareFunc {
	return func(n janice.HandlerFunc) janice.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) error {
			pfn := strudel.NextID
			defer func() {
				strudel.NextID = pfn
			}()
			strudel.NextID = func() string {
				return v
			}
			return n(w, r)
		}
	}
}
