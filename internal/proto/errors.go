package proto

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Sentinel errors returned by Client methods so callers can switch on
// them via errors.Is without parsing strings.
var (
	ErrSessionNotFound   = errors.New("session not found")
	ErrSessionLocked     = errors.New("session locked")
	ErrWrongPassword     = errors.New("wrong password")
	ErrUnauthorized      = errors.New("not authenticated")
	ErrTooManyRequests   = errors.New("too many requests")
	ErrFileTooLarge      = errors.New("file too large")
	ErrServerUnavailable = errors.New("server unavailable")
	ErrDiskFull          = errors.New("server storage exhausted")
)

// HTTPError wraps any non-success response that doesn't map to a more
// specific sentinel. Status is the HTTP status code; Detail is the
// "detail" field of the server's JSON error body, if present.
type HTTPError struct {
	Status int
	Detail string
}

func (e *HTTPError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("http %d: %s", e.Status, e.Detail)
	}
	return fmt.Sprintf("http %d", e.Status)
}

// parseHTTPError reads the response body, attempts to parse it as the
// server's standard {"detail": "..."} error JSON, and returns the
// appropriate sentinel (or a generic HTTPError).
func parseHTTPError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	detail := extractDetail(body)
	sentinel := sentinelFor(resp.StatusCode, detail)
	if sentinel != nil {
		if detail == "" {
			return sentinel
		}
		return fmt.Errorf("%w: %s", sentinel, detail)
	}
	return &HTTPError{Status: resp.StatusCode, Detail: detail}
}

func extractDetail(body []byte) string {
	var env struct {
		Detail string `json:"detail"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return ""
	}
	if env.Detail != "" {
		return env.Detail
	}
	return env.Error
}

func sentinelFor(status int, detail string) error {
	switch status {
	case http.StatusNotFound:
		return ErrSessionNotFound
	case http.StatusGone:
		return ErrSessionLocked
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusTooManyRequests:
		return ErrTooManyRequests
	case http.StatusRequestEntityTooLarge:
		return ErrFileTooLarge
	case http.StatusServiceUnavailable:
		return ErrServerUnavailable
	case http.StatusInsufficientStorage:
		return ErrDiskFull
	}
	return nil
}

// standardStatusErrors maps a successful (2xx) response to nil and any
// other status to the right sentinel or a generic HTTPError. Called by
// every method whose response body the caller doesn't need on error.
func standardStatusErrors(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return parseHTTPError(resp)
}
