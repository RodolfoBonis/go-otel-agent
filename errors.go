package otelagent

import "errors"

var (
	ErrNotInitialized     = errors.New("go-otel-agent: agent not initialized, call Init() first")
	ErrAlreadyInitialized = errors.New("go-otel-agent: agent already initialized")
	ErrInvalidConfig      = errors.New("go-otel-agent: invalid configuration")
	ErrShutdownTimeout    = errors.New("go-otel-agent: shutdown timed out")
	ErrMissingServiceName = errors.New("go-otel-agent: service name is required")
)
