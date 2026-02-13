package provider

import (
	"context"
	"regexp"
	"strings"
	"sync"

	"github.com/RodolfoBonis/go-otel-agent/config"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// ScrubProcessor is a SpanProcessor that redacts PII from span attributes
// before they are exported. Applied at attribute-setting level since
// ReadOnlySpan is immutable after span end.
type ScrubProcessor struct {
	config           config.ScrubConfig
	sensitiveKeys    map[string]struct{}
	compiledPatterns []*regexp.Regexp
	once             sync.Once
}

// NewScrubProcessor creates a new PII scrubbing span processor.
func NewScrubProcessor(cfg config.ScrubConfig) *ScrubProcessor {
	sp := &ScrubProcessor{
		config:        cfg,
		sensitiveKeys: make(map[string]struct{}),
	}
	sp.init()
	return sp
}

func (sp *ScrubProcessor) init() {
	sp.once.Do(func() {
		for _, key := range sp.config.SensitiveKeys {
			sp.sensitiveKeys[key] = struct{}{}
		}

		for _, pattern := range sp.config.SensitivePatterns {
			if re, err := regexp.Compile(pattern); err == nil {
				sp.compiledPatterns = append(sp.compiledPatterns, re)
			}
		}
	})
}

// OnStart is called when a span starts. We scrub attributes here since
// we can still modify the span.
func (sp *ScrubProcessor) OnStart(_ context.Context, s sdktrace.ReadWriteSpan) {
	if !sp.config.Enabled {
		return
	}

	redacted := sp.config.RedactedValue
	if redacted == "" {
		redacted = "[REDACTED]"
	}

	attrs := s.Attributes()
	var scrubbed []attribute.KeyValue

	for _, attr := range attrs {
		key := string(attr.Key)

		if sp.isSensitive(key) {
			scrubbed = append(scrubbed, attribute.String(key, redacted))
		}
	}

	if len(scrubbed) > 0 {
		s.SetAttributes(scrubbed...)
	}

	// DB statement truncation (separate concern from PII redaction)
	if sp.config.DBStatementMaxLength > 0 {
		sp.truncateDBStatements(s)
	}
}

// truncateDBStatements applies length truncation to DB query attributes.
// Handles both db.statement (legacy semconv) and db.query.text (new semconv).
func (sp *ScrubProcessor) truncateDBStatements(s sdktrace.ReadWriteSpan) {
	dbKeys := []string{"db.statement", "db.query.text"}
	attrs := s.Attributes()
	var truncated []attribute.KeyValue

	for _, attr := range attrs {
		key := string(attr.Key)
		for _, dbKey := range dbKeys {
			if key == dbKey {
				val := attr.Value.AsString()
				if len(val) > sp.config.DBStatementMaxLength {
					truncated = append(truncated, attribute.String(key, val[:sp.config.DBStatementMaxLength]+"..."))
				}
			}
		}
	}

	if len(truncated) > 0 {
		s.SetAttributes(truncated...)
	}
}

// OnEnd is called when a span ends.
func (sp *ScrubProcessor) OnEnd(_ sdktrace.ReadOnlySpan) {}

// Shutdown shuts down the processor.
func (sp *ScrubProcessor) Shutdown(_ context.Context) error { return nil }

// ForceFlush forces a flush of the processor.
func (sp *ScrubProcessor) ForceFlush(_ context.Context) error { return nil }

func (sp *ScrubProcessor) isSensitive(key string) bool {
	if _, ok := sp.sensitiveKeys[key]; ok {
		return true
	}

	lowerKey := strings.ToLower(key)
	for _, re := range sp.compiledPatterns {
		if re.MatchString(lowerKey) {
			return true
		}
	}

	return false
}
