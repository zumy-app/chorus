package translation

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Cooldown durations applied when a provider returns an error that warrants
// pausing it (rate limit, quota, auth, transient server error).
const (
	// cooldownRateLimit pauses a provider briefly after a 429.
	cooldownRateLimit = 60 * time.Second
	// cooldownServerError pauses a provider briefly after a 5xx.
	cooldownServerError = 30 * time.Second
	// cooldownAuth pauses a provider for a while after 401/403, since the
	// credentials will not start working again on their own.
	cooldownAuth = 15 * time.Minute
)

// HTTPStatusError represents a non-2xx HTTP response from a translation
// provider. It carries enough detail for the chain to classify the failure
// (rate limit vs. auth vs. server error) and apply an appropriate cooldown.
type HTTPStatusError struct {
	Provider   string
	StatusCode int
	Body       string
}

// Error implements the error interface.
func (e *HTTPStatusError) Error() string {
	msg := strings.TrimSpace(e.Body)
	if msg == "" {
		msg = http.StatusText(e.StatusCode)
	}
	return fmt.Sprintf("%s returned HTTP %d: %s", e.Provider, e.StatusCode, msg)
}

// NewHTTPStatusError creates an error describing a non-2xx response from a
// translation provider.
func NewHTTPStatusError(provider string, statusCode int, body string) error {
	return &HTTPStatusError{Provider: provider, StatusCode: statusCode, Body: body}
}

// CooldownFor returns how long a provider should be paused after the given
// error. It returns 0 (or negative) for errors that should not warrant a pause.
//
// Rate limits (429), exhausted quotas / auth failures (401/403) and transient
// server errors (5xx) put the provider into cooldown; everything else is left
// for the chain to simply fall through on the next request.
func CooldownFor(err error) time.Duration {
	var hse *HTTPStatusError
	if errors.As(err, &hse) {
		switch {
		case hse.StatusCode == http.StatusTooManyRequests:
			return cooldownRateLimit
		case hse.StatusCode == http.StatusUnauthorized || hse.StatusCode == http.StatusForbidden:
			return cooldownAuth
		case hse.StatusCode >= 500:
			return cooldownServerError
		default:
			return 0
		}
	}
	return 0
}

// IsModelNotFound reports whether err indicates the requested model is not
// installed on the provider. Ollama returns HTTP 404 with a body such as
// {"error":"model 'qwen2.5:3b' not found"} when the model is missing; the
// check also covers non-HTTP errors that mention a missing model.
func IsModelNotFound(err error) bool {
	if err == nil {
		return false
	}
	var hse *HTTPStatusError
	if errors.As(err, &hse) {
		if hse.StatusCode == http.StatusNotFound && strings.Contains(strings.ToLower(hse.Body), "not found") {
			return true
		}
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "model") && strings.Contains(lower, "not found")
}
