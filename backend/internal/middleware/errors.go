package middleware

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Kind classifies an API error so clients can render the right UI (a field-level
// validation hint vs. a retryable toast) and decide whether the same request may
// safely be retried.
type Kind string

const (
	// KindValidation: the request payload is malformed or fails a rule.
	KindValidation Kind = "validation"
	// KindAuth: the caller is missing/expired credentials.
	KindAuth Kind = "auth"
	// KindForbidden: the caller is authenticated but not allowed to act.
	KindForbidden Kind = "forbidden"
	// KindNotFound: the requested resource does not exist.
	KindNotFound Kind = "not_found"
	// KindConflict: the request conflicts with current state (e.g. duplicate).
	KindConflict Kind = "conflict"
	// KindRateLimit: the caller exceeded a rate quota.
	KindRateLimit Kind = "rate_limit"
	// KindTranslation: an upstream translation provider failed.
	KindTranslation Kind = "translation"
	// KindTooLarge: the request exceeds a size budget.
	KindTooLarge Kind = "too_large"
	// KindUnavailable: a required dependency is not configured or temporarily
	// unavailable (e.g. payments not configured, provider queue down).
	KindUnavailable Kind = "unavailable"
	// KindDelivery: a best-effort asynchronous delivery (e.g. email) could not
	// complete synchronously but has been queued for automatic retry.
	KindDelivery Kind = "delivery"
	// KindInternal: an unexpected server error. Details are logged, not leaked.
	KindInternal Kind = "internal"
)

// Status maps a Kind to the HTTP status code a client should receive.
func (k Kind) Status() int {
	switch k {
	case KindValidation:
		return http.StatusBadRequest
	case KindAuth:
		return http.StatusUnauthorized
	case KindForbidden:
		return http.StatusForbidden
	case KindNotFound:
		return http.StatusNotFound
	case KindConflict:
		return http.StatusConflict
	case KindRateLimit:
		return http.StatusTooManyRequests
	case KindTranslation:
		return http.StatusBadGateway
	case KindTooLarge:
		return http.StatusRequestEntityTooLarge
	case KindUnavailable:
		return http.StatusServiceUnavailable
	case KindDelivery:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// retryByDefault reports whether a client may blanket-retry an error of this
// Kind without special handling (validation/auth errors are not retryable).
func (k Kind) retryByDefault() bool {
	switch k {
	case KindRateLimit, KindTranslation, KindTooLarge, KindInternal, KindUnavailable, KindDelivery:
		return true
	default:
		return false
	}
}

// APIError is the single concrete error type returned across the HTTP surface.
// It carries a Kind for classification, a client-safe Message, an optional
// wrapped Cause for logging/tracing, and a Retryable flag.
type APIError struct {
	Kind      Kind
	Message   string
	Cause     error
	Retryable bool
}

// Error implements the error interface. The cause is only surfaced when the
// message is empty, so clients never see raw internals by accident.
func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return "an error occurred"
}

// Unwrap exposes the wrapped cause for errors.Is / errors.As.
func (e *APIError) Unwrap() error { return e.Cause }

// NewError builds an APIError of the given Kind. Retryable defaults from the
// Kind unless overridden via WithRetryable.
func NewError(kind Kind, message string) *APIError {
	return &APIError{Kind: kind, Message: message, Retryable: kind.retryByDefault()}
}

// Errorf builds an APIError of the given Kind with a formatted client message.
func Errorf(kind Kind, format string, args ...any) *APIError {
	return &APIError{Kind: kind, Message: fmt.Sprintf(format, args...), Retryable: kind.retryByDefault()}
}

// WithRetryable returns a copy of e with the retryable flag overridden.
func (e *APIError) WithRetryable(retryable bool) *APIError {
	if e == nil {
		return nil
	}
	cp := *e
	cp.Retryable = retryable
	return &cp
}

// WithCause returns a copy of e that wraps cause for logging while keeping the
// client-safe message.
func (e *APIError) WithCause(cause error) *APIError {
	if e == nil {
		return nil
	}
	cp := *e
	cp.Cause = cause
	return &cp
}

// Convenience constructors for the Kinds the client is expected to special-case.
func ErrValidation(message string) *APIError  { return NewError(KindValidation, message) }
func ErrAuth(message string) *APIError        { return NewError(KindAuth, message) }
func ErrForbidden(message string) *APIError   { return NewError(KindForbidden, message) }
func ErrNotFound(message string) *APIError    { return NewError(KindNotFound, message) }
func ErrConflict(message string) *APIError    { return NewError(KindConflict, message) }
func ErrRateLimit(message string) *APIError   { return NewError(KindRateLimit, message) }
func ErrTranslation(message string) *APIError { return NewError(KindTranslation, message) }
func ErrTooLarge(message string) *APIError    { return NewError(KindTooLarge, message) }
func ErrUnavailable(message string) *APIError { return NewError(KindUnavailable, message) }
func ErrDelivery(message string) *APIError    { return NewError(KindDelivery, message) }
func ErrInternal(message string) *APIError    { return NewError(KindInternal, message) }

// FromErr normalizes any error into an *APIError:
//   - nil        -> an internal error with a generic message
//   - *APIError  -> returned unchanged
//   - anything else -> wrapped as an internal error that retains the original
//     as its Cause for logging (never leaked to the client).
func FromErr(err error) *APIError {
	if err == nil {
		return &APIError{Kind: KindInternal, Message: "Something went wrong", Retryable: true}
	}
	var api *APIError
	if errors.As(err, &api) {
		return api
	}
	return &APIError{
		Kind:      KindInternal,
		Message:   "Something went wrong",
		Cause:     err,
		Retryable: true,
	}
}

// KindOf returns the classification of err, or ok=false when err is not an
// *APIError (e.g. a plain service error).
func KindOf(err error) (Kind, bool) {
	if err == nil {
		return "", false
	}
	var api *APIError
	if errors.As(err, &api) {
		return api.Kind, true
	}
	return "", false
}

// StatusFor resolves the HTTP status code for err, defaulting to 500 for
// untyped errors.
func StatusFor(err error) int {
	return FromErr(err).Kind.Status()
}

// IsRetryable reports whether a client may safely retry the same request. It
// returns false for untyped errors.
func IsRetryable(err error) bool {
	var api *APIError
	if errors.As(err, &api) {
		return api.Retryable
	}
	return false
}

// WriteError terminates the request with a structured, typed error envelope:
//
//	{ "error": { "kind": "...", "message": "...", "retryable": bool } }
//
// Unexpected (internal) errors are logged with their cause and always return the
// generic message so raw internals never reach the client.
func WriteError(c *gin.Context, err error) {
	api := FromErr(err)
	if api.Kind == KindInternal && api.Cause != nil {
		log.Printf("internal error (request %s %s): %v", c.Request.Method, c.Request.URL.Path, api.Cause)
	}
	c.AbortWithStatusJSON(api.Kind.Status(), gin.H{
		"error": gin.H{
			"kind":      api.Kind,
			"message":   api.Message,
			"retryable": api.Retryable,
		},
	})
}

// Recovery converts a panicking handler into the same typed internal-error
// envelope used everywhere else, so even an unexpected panic never leaks a raw
// stack trace or an empty body. Use it in place of gin.Recovery().
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic (request %s %s): %v\n%s", c.Request.Method, c.Request.URL.Path, rec, debug.Stack())
				WriteError(c, &APIError{
					Kind:      KindInternal,
					Message:   "Something went wrong",
					Retryable: true,
				})
			}
		}()
		c.Next()
	}
}
