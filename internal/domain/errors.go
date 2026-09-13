package domain

import "errors"

var (
	// ErrUserNotFound is returned when a requested user does not exist.
	ErrUserNotFound = errors.New("user not found")

	// ErrUserAlreadyExists is returned when attempting to register an existing user.
	ErrUserAlreadyExists = errors.New("user already exists")

	// ErrInvalidToken is returned when a JWT or OAuth token is invalid or expired.
	ErrInvalidToken = errors.New("invalid or expired token")

	// ErrUnauthorized is returned when an unauthenticated request attempts to access a protected resource.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrInvalidInput is returned when client input fails domain validation.
	ErrInvalidInput = errors.New("invalid input")

	// ErrGoogleAuthFailed is returned when verification with Google services fails.
	ErrGoogleAuthFailed = errors.New("google authentication failed")

	// ErrLLMExecutionFailed is returned when the LLM service fails to complete generation.
	ErrLLMExecutionFailed = errors.New("llm generation failed")

	// ErrDatabase is returned on unhandled persistence errors.
	ErrDatabase = errors.New("database operation failed")
)
