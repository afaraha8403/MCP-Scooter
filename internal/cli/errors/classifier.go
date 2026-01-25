package errors

import (
	"strings"
)

type ErrorKind string

const (
	ErrorKindAuth     ErrorKind = "auth"
	ErrorKindOffline  ErrorKind = "offline"
	ErrorKindHTTP     ErrorKind = "http"
	ErrorKindStdio    ErrorKind = "stdio-exit"
	ErrorKindNotFound ErrorKind = "not-found"
	ErrorKindOther    ErrorKind = "other"
)

type ClassifiedError struct {
	Kind    ErrorKind
	Message string
	Hint    string // User-friendly suggestion
	Raw     error
}

func (e ClassifiedError) Error() string {
	return e.Message
}

func Classify(err error) ClassifiedError {
	if err == nil {
		return ClassifiedError{}
	}

	msg := strings.ToLower(err.Error())
	
	switch {
	case strings.Contains(msg, "401") || strings.Contains(msg, "unauthorized") || strings.Contains(msg, "invalid_token"):
		return ClassifiedError{
			Kind:    ErrorKindAuth,
			Message: err.Error(),
			Hint:    "Check your API key or run 'scooter auth'",
			Raw:     err,
		}
	case strings.Contains(msg, "connection refused") || strings.Contains(msg, "timeout") || strings.Contains(msg, "fetch failed") || strings.Contains(msg, "econnrefused"):
		return ClassifiedError{
			Kind:    ErrorKindOffline,
			Message: err.Error(),
			Hint:    "Is the Scooter daemon running? Try 'scooter status' or start it with 'scooter'",
			Raw:     err,
		}
	case strings.Contains(msg, "404") || strings.Contains(msg, "not found"):
		return ClassifiedError{
			Kind:    ErrorKindNotFound,
			Message: err.Error(),
			Hint:    "The requested resource was not found. Check the server or tool name.",
			Raw:     err,
		}
	case strings.Contains(msg, "exit status") || strings.Contains(msg, "signal:"):
		return ClassifiedError{
			Kind:    ErrorKindStdio,
			Message: err.Error(),
			Hint:    "The MCP server process exited unexpectedly. Check the server logs.",
			Raw:     err,
		}
	case strings.Contains(msg, "http"):
		return ClassifiedError{
			Kind:    ErrorKindHTTP,
			Message: err.Error(),
			Hint:    "An HTTP error occurred during communication with the daemon.",
			Raw:     err,
		}
	default:
		return ClassifiedError{
			Kind:    ErrorKindOther,
			Message: err.Error(),
			Hint:    "An unexpected error occurred.",
			Raw:     err,
		}
	}
}
