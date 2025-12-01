package domain

import "errors"

// LM Studio error types

var (
	// ErrLMStudioUnavailable indicates the LM Studio service is unavailable
	ErrLMStudioUnavailable = errors.New("lm studio service unavailable")

	// ErrLMStudioTimeout indicates a request to LM Studio timed out
	ErrLMStudioTimeout = errors.New("lm studio request timeout")

	// ErrInvalidRequest indicates an invalid request was made (4xx client errors)
	ErrInvalidRequest = errors.New("invalid request")
)
