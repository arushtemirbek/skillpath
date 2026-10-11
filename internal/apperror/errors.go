// Package apperror defines application errors independently of HTTP transport.
package apperror

import "errors"

var (
	ErrGoalRequired        = errors.New("goal is required")
	ErrInvalidRequest      = errors.New("invalid request")
	ErrUnprocessableEntity = errors.New("unprocessable entity")
	ErrLLMRequestFailed    = errors.New("LLM request failed")
	ErrLLMInvalidResponse  = errors.New("LLM returned an invalid response")
)
