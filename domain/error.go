package domain

import (
	stderr "errors"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

// ErrRecordNotFound is used to make our application logic independent of other libraries errors
var ErrRecordNotFound = errors.New("record not found")

// Common errors used across the application
// These errors are used to provide consistent error handling and responses
var (
	ErrNotFound = DetailedError{
		IDField:         "NOT_FOUND",
		StatusDescField: http.StatusText(http.StatusNotFound),
		ErrorField:      "The requested resource could not be found",
		StatusCodeField: http.StatusNotFound,
	}

	ErrUnauthorized = DetailedError{
		IDField:         "UNAUTHORIZED",
		StatusDescField: http.StatusText(http.StatusUnauthorized),
		ErrorField:      "The request could not be authorized",
		StatusCodeField: http.StatusUnauthorized,
	}

	ErrForbidden = DetailedError{
		IDField:         "FORBIDDEN",
		StatusDescField: http.StatusText(http.StatusForbidden),
		ErrorField:      "The requested action was forbidden",
		StatusCodeField: http.StatusForbidden,
	}

	ErrTooManyRequests = DetailedError{
		IDField:         "TOO_MANY_REQUESTS",
		StatusDescField: http.StatusText(http.StatusTooManyRequests),
		ErrorField:      "Too many requests, please try again later",
		StatusCodeField: http.StatusTooManyRequests,
	}

	ErrInternalServerError = DetailedError{
		IDField:         "INTERNAL_SERVER_ERROR",
		StatusDescField: http.StatusText(http.StatusInternalServerError),
		ErrorField:      "An internal server error occurred, please contact the system administrator",
		StatusCodeField: http.StatusInternalServerError,
	}

	ErrBadRequest = DetailedError{
		IDField:         "BAD_REQUEST",
		StatusDescField: http.StatusText(http.StatusBadRequest),
		ErrorField:      "The request was malformed or contained invalid parameters",
		StatusCodeField: http.StatusBadRequest,
	}

	ErrUnsupportedMediaType = DetailedError{
		IDField:         "UNSUPPORTED_MEDIA_TYPE",
		StatusDescField: http.StatusText(http.StatusUnsupportedMediaType),
		ErrorField:      "The request is using an unknown content type",
		StatusCodeField: http.StatusUnsupportedMediaType,
	}

	ErrConflict = DetailedError{
		IDField:         "CONFLICT",
		StatusDescField: http.StatusText(http.StatusConflict),
		ErrorField:      "The resource could not be created due to a conflict",
		StatusCodeField: http.StatusConflict,
	}

	ErrNotImplemented = DetailedError{
		IDField:         "NOT_IMPLEMENTED",
		StatusDescField: http.StatusText(http.StatusInternalServerError),
		ErrorField:      "This feature is not implemented yet",
		StatusCodeField: http.StatusInternalServerError,
	}
)

type DetailedError struct {
	// The error ID
	//
	// Useful when trying to identify various errors in application logic.
	IDField string `json:"id,omitempty"`

	// The status code
	//
	// example: 404
	StatusCodeField int `json:"code,omitempty"`

	// The status description
	//
	// example: Not Found
	StatusDescField string `json:"status,omitempty"`

	// The request ID
	//
	// The request ID is often exposed internally in order to trace
	// errors across service architectures. This is often a UUID.
	//
	// example: d7ef54b1-ec15-46e6-bccb-524b82c035e6
	RIDField string `json:"request,omitempty"`

	// A human-readable reason for the error
	//
	// example: User with ID 1234 does not exist.
	ReasonField string `json:"reason,omitempty"`

	// Debug information
	//
	// This field is often not exposed to protect against leaking
	// sensitive information.
	//
	// example: SQL field "foo" is not a bool.
	DebugField string `json:"debug,omitempty"`

	// Error message
	//
	// The error's message.
	//
	// example: The resource could not be found
	// required: true
	ErrorField string `json:"message"`

	// Further error details
	DetailsField map[string]interface{} `json:"details,omitempty"`

	err error
}

// StackTrace returns the error's stack trace.
func (e *DetailedError) StackTrace() (trace errors.StackTrace) {
	if e.err == e {
		return
	}

	if st := stackTracer(nil); stderr.As(e.err, &st) {
		trace = st.StackTrace()
	}

	return
}

func (e DetailedError) Unwrap() error {
	return e.err
}

func (e *DetailedError) Wrap(err error) {
	e.err = err
}

func (e DetailedError) WithWrap(err error) *DetailedError {
	e.err = err
	return &e
}

func (e DetailedError) WithID(id string) *DetailedError {
	e.IDField = id
	return &e
}

func (e *DetailedError) WithTrace(err error) *DetailedError {
	if st := stackTracer(nil); !stderr.As(e.err, &st) {
		e.Wrap(errors.WithStack(err))
	} else {
		e.Wrap(err)
	}
	return e
}

func (e DetailedError) Is(err error) bool {
	switch te := err.(type) {
	case DetailedError:
		return e.ErrorField == te.ErrorField &&
			e.StatusDescField == te.StatusDescField &&
			e.IDField == te.IDField &&
			e.StatusCodeField == te.StatusCodeField
	case *DetailedError:
		return e.ErrorField == te.ErrorField &&
			e.StatusDescField == te.StatusDescField &&
			e.IDField == te.IDField &&
			e.StatusCodeField == te.StatusCodeField
	default:
		return false
	}
}

func (e DetailedError) Status() string {
	return e.StatusDescField
}

func (e DetailedError) ID() string {
	return e.IDField
}

func (e DetailedError) Error() string {
	return e.ErrorField
}

func (e DetailedError) RequestID() string {
	return e.RIDField
}

func (e DetailedError) Reason() string {
	return e.ReasonField
}

func (e DetailedError) Debug() string {
	return e.DebugField
}

func (e DetailedError) Details() map[string]interface{} {
	return e.DetailsField
}

func (e DetailedError) StatusCode() int {
	return e.StatusCodeField
}

func (e DetailedError) WithReason(reason string) *DetailedError {
	e.ReasonField = reason
	return &e
}

func (e DetailedError) WithReasonf(reason string, args ...interface{}) *DetailedError {
	return e.WithReason(fmt.Sprintf(reason, args...))
}

func (e DetailedError) WithError(message string) *DetailedError {
	e.ErrorField = message
	return &e
}

func (e DetailedError) WithErrorf(message string, args ...interface{}) *DetailedError {
	return e.WithError(fmt.Sprintf(message, args...))
}

func (e DetailedError) WithDebugf(debug string, args ...interface{}) *DetailedError {
	return e.WithDebug(fmt.Sprintf(debug, args...))
}

func (e DetailedError) WithDebug(debug string) *DetailedError {
	e.DebugField = debug
	return &e
}

func (e DetailedError) WithDetail(key string, detail interface{}) *DetailedError {
	if e.DetailsField == nil {
		e.DetailsField = map[string]interface{}{}
	}
	e.DetailsField[key] = detail
	return &e
}

func (e DetailedError) WithDetailf(key string, message string, args ...interface{}) *DetailedError {
	if e.DetailsField == nil {
		e.DetailsField = map[string]interface{}{}
	}
	e.DetailsField[key] = fmt.Sprintf(message, args...)
	return &e
}

func (e DetailedError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = fmt.Fprintf(s, "id=%s\n", e.IDField)
			_, _ = fmt.Fprintf(s, "rid=%s\n", e.RIDField)
			_, _ = fmt.Fprintf(s, "error=%s\n", e.ErrorField)
			_, _ = fmt.Fprintf(s, "reason=%s\n", e.ReasonField)
			_, _ = fmt.Fprintf(s, "details=%+v\n", e.DetailsField)
			_, _ = fmt.Fprintf(s, "debug=%s\n", e.DebugField)
			e.StackTrace().Format(s, verb)
			return
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, e.ErrorField)
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", e.ErrorField)
	}
}

func ToDefaultError(err error, requestID string) *DetailedError {
	de := &DetailedError{
		RIDField:        requestID,
		StatusCodeField: http.StatusInternalServerError,
		DetailsField:    map[string]interface{}{},
		ErrorField:      err.Error(),
	}
	de.Wrap(err)

	if c := ReasonCarrier(nil); stderr.As(err, &c) {
		de.ReasonField = c.Reason()
	}
	if c := RequestIDCarrier(nil); stderr.As(err, &c) && c.RequestID() != "" {
		de.RIDField = c.RequestID()
	}
	if c := DetailsCarrier(nil); stderr.As(err, &c) && c.Details() != nil {
		de.DetailsField = c.Details()
	}
	if c := StatusCarrier(nil); stderr.As(err, &c) && c.Status() != "" {
		de.StatusDescField = c.Status()
	}
	if c := StatusCodeCarrier(nil); stderr.As(err, &c) && c.StatusCode() != 0 {
		de.StatusCodeField = c.StatusCode()
	}
	if c := DebugCarrier(nil); stderr.As(err, &c) {
		de.DebugField = c.Debug()
	}
	if c := IDCarrier(nil); stderr.As(err, &c) {
		de.IDField = c.ID()
	}

	if de.StatusDescField == "" {
		de.StatusDescField = http.StatusText(de.StatusCode())
	}

	return de
}

// StatusCodeCarrier can be implemented by an error to support setting status codes in the error itself.
type StatusCodeCarrier interface {
	// StatusCode returns the status code of this error.
	StatusCode() int
}

// RequestIDCarrier can be implemented by an error to support error contexts.
type RequestIDCarrier interface {
	// RequestID returns the ID of the request that caused the error, if applicable.
	RequestID() string
}

// ReasonCarrier can be implemented by an error to support error contexts.
type ReasonCarrier interface {
	// Reason returns the reason for the error, if applicable.
	Reason() string
}

// DebugCarrier can be implemented by an error to support error contexts.
type DebugCarrier interface {
	// Debug returns debugging information for the error, if applicable.
	Debug() string
}

// StatusCarrier can be implemented by an error to support error contexts.
type StatusCarrier interface {
	// ID returns the error id, if applicable.
	Status() string
}

// DetailsCarrier can be implemented by an error to support error contexts.
type DetailsCarrier interface {
	// Details returns details on the error, if applicable.
	Details() map[string]interface{}
}

// IDCarrier can be implemented by an error to support error contexts.
type IDCarrier interface {
	// ID returns application error ID on the error, if applicable.
	ID() string
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}
