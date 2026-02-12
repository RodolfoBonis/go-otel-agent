package logger

import "context"

// NoopLogger is a logger that does nothing. Useful for testing.
type NoopLogger struct{}

var _ Logger = (*NoopLogger)(nil)

func (n *NoopLogger) Debug(_ context.Context, _ string, _ ...Fields)   {}
func (n *NoopLogger) Info(_ context.Context, _ string, _ ...Fields)    {}
func (n *NoopLogger) Warning(_ context.Context, _ string, _ ...Fields) {}
func (n *NoopLogger) Error(_ context.Context, _ string, _ ...Fields)   {}
func (n *NoopLogger) Fatal(_ context.Context, _ string, _ ...Fields)   {}
func (n *NoopLogger) Panic(_ context.Context, _ string, _ ...Fields)   {}
func (n *NoopLogger) With(_ Fields) Logger                             { return n }
func (n *NoopLogger) LogError(_ context.Context, _ string, _ error)    {}
