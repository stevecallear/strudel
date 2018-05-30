package strudel

import (
	"context"
	"net/http"

	"github.com/felixge/httpsnoop"
	"github.com/gamegos/jsend"
	"github.com/sirupsen/logrus"
	"github.com/stevecallear/janice"

	uuid "github.com/satori/go.uuid"
)

var (
	// Logger is the logger used for all middleware
	Logger *logrus.Logger

	// GetRequestID returns the id for the specified request
	GetRequestID = func(r *http.Request) (string, bool) {
		v, _ := r.Context().Value(reqIDKey).(string)
		return v, v != ""
	}

	reqIDKey = contextKey("requestid")
)

type contextKey string

func init() {
	Logger = logrus.New()
	Logger.Formatter = new(logrus.JSONFormatter)
}

// RequestTracking is a request tracking middleware function
func RequestTracking(n janice.HandlerFunc) janice.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		id := uuid.Must(uuid.NewV4()).String()
		ctx := context.WithValue(r.Context(), reqIDKey, id)
		return n(w, r.WithContext(ctx))
	}
}

// RequestLogging is a request logging middleware function
func RequestLogging(n janice.HandlerFunc) janice.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		p := r.URL.String()
		var err error
		m := httpsnoop.CaptureMetricsFn(w, func(ww http.ResponseWriter) {
			err = n(ww, r)
		})
		le := Logger.WithFields(logrus.Fields{
			"type":     "request",
			"host":     r.Host,
			"method":   r.Method,
			"path":     p,
			"code":     m.Code,
			"duration": m.Duration.String(),
			"written":  m.Written,
		})
		if rid, ok := GetRequestID(r); ok {
			le = le.WithField("request", rid)
		}
		le.Info()
		return err
	}
}

// Recovery is a panic recovery middleware function
func Recovery(n janice.HandlerFunc) janice.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		defer func() {
			if rec := recover(); rec != nil {
				le := Logger.WithField("type", "recovery")
				if rid, ok := GetRequestID(r); ok {
					le = le.WithField("request", rid)
				}
				le.Error(rec)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		return n(w, r)
	}
}

// ErrorHandling is an error handling middleware function
func ErrorHandling(n janice.HandlerFunc) janice.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := n(w, r); err != nil {
			le := Logger.WithField("type", "error")
			jw := jsend.Wrap(w).
				Status(http.StatusInternalServerError).
				Message(http.StatusText(http.StatusInternalServerError))
			if err, ok := err.(*Error); ok {
				c := err.Code()
				if c > 0 {
					le = le.WithField("code", c)
				}
				if c >= 400 && c < 600 {
					jw = jw.Status(c)
				}
				if f := err.Fields(); len(f) > 0 {
					jw = jw.Data(f)
				}
				if lf := err.LogFields(); len(lf) > 0 {
					le = le.WithField("data", lf)
				}
				jw = jw.Message(err.Error())
			}
			if rid, ok := GetRequestID(r); ok {
				le = le.WithField("request", rid)
			}
			le.Error(err.Error())
			_, err := jw.Send()
			return err
		}
		return nil
	}
}
