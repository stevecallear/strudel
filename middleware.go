package strudel

import (
	"context"
	"net/http"
	"strconv"

	"github.com/felixge/httpsnoop"
	"github.com/gamegos/jsend"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/stevecallear/janice"
)

var (
	// Logger is the logger used for all middleware
	Logger *logrus.Logger

	// NextID returns the next unique id string
	NextID = func() string {
		return xid.New().String()
	}

	reqIDKey = contextKey("reqID")
)

type contextKey string

func init() {
	Logger = logrus.New()
	Logger.Formatter = new(logrus.JSONFormatter)
}

// RequestLogging is a request logging middleware function
func RequestLogging(n janice.HandlerFunc) janice.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		p := r.URL.String()
		rid := NextID()
		var err error
		m := httpsnoop.CaptureMetricsFn(w, func(ww http.ResponseWriter) {
			err = n(ww, setReqID(r, rid))
		})
		Logger.WithFields(logrus.Fields{
			"type":     "request",
			"request":  rid,
			"host":     r.Host,
			"method":   r.Method,
			"path":     p,
			"code":     strconv.Itoa(m.Code),
			"duration": m.Duration.String(),
			"written":  strconv.FormatInt(m.Written, 10),
		}).Info()
		return err
	}
}

// Recovery is a panic recovery middleware function
func Recovery(n janice.HandlerFunc) janice.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		defer func() {
			if rec := recover(); rec != nil {
				le := Logger.WithField("type", "recovery")
				if rid := getReqID(r); rid != "" {
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
			le.Error(err.Error())
			_, err := jw.Send()
			return err
		}
		return nil
	}
}

func setReqID(r *http.Request, v string) *http.Request {
	ctx := context.WithValue(r.Context(), reqIDKey, v)
	return r.WithContext(ctx)
}

func getReqID(r *http.Request) string {
	v, _ := r.Context().Value(reqIDKey).(string)
	return v
}
