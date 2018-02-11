package strudel

import (
	"net/http"
	"strconv"

	"github.com/felixge/httpsnoop"
	"github.com/gamegos/jsend"
	"github.com/sirupsen/logrus"
	"github.com/stevecallear/janice"
)

// Logger is the logger used for all middleware
var Logger *logrus.Logger

func init() {
	Logger = logrus.New()
	Logger.Formatter = new(logrus.JSONFormatter)
}

// RequestLogging is a request logging middleware function
func RequestLogging(n janice.HandlerFunc) janice.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		p := r.URL.String()
		var err error
		m := httpsnoop.CaptureMetricsFn(w, func(ww http.ResponseWriter) {
			err = n(ww, r)
		})
		Logger.WithFields(logrus.Fields{
			"type":     "request",
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

// Recovery is panic recovery middleware function
func Recovery(n janice.HandlerFunc) janice.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		defer func() {
			if r := recover(); r != nil {
				Logger.WithField("type", "recovery").Error(r)
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
			jw := jsend.Wrap(w).Status(http.StatusInternalServerError).Message(err.Error())
			if serr, ok := err.(*Error); ok {
				if serr.Code() >= 400 && serr.Code() < 600 {
					jw.Status(serr.Code())
				}
				if f := serr.Fields(); len(f) > 0 {
					le = le.WithField("data", f)
					jw.Data(f)
				}
			}
			le.Error(err.Error())
			_, serr := jw.Send()
			return serr
		}
		return nil
	}
}
