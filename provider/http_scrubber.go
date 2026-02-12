package provider

import (
	"regexp"
	"strings"
	"sync"

	"github.com/RodolfoBonis/go-otel-agent/config"
)

// HTTPScrubber provides PII scrubbing for HTTP request/response data.
type HTTPScrubber struct {
	httpCfg  config.HTTPConfig
	scrubCfg config.ScrubConfig

	sensitiveHeaderSet map[string]struct{}
	compiledPatterns   []*regexp.Regexp
	allowedContentSet  map[string]struct{}
	once               sync.Once
}

// NewHTTPScrubber creates an HTTP scrubber from HTTP and scrub configurations.
func NewHTTPScrubber(httpCfg config.HTTPConfig, scrubCfg config.ScrubConfig) *HTTPScrubber {
	s := &HTTPScrubber{
		httpCfg:  httpCfg,
		scrubCfg: scrubCfg,
	}
	s.init()
	return s
}

func (s *HTTPScrubber) init() {
	s.once.Do(func() {
		s.sensitiveHeaderSet = make(map[string]struct{}, len(s.httpCfg.SensitiveHeaders))
		for _, h := range s.httpCfg.SensitiveHeaders {
			s.sensitiveHeaderSet[strings.ToLower(h)] = struct{}{}
		}

		s.allowedContentSet = make(map[string]struct{}, len(s.httpCfg.BodyAllowedContentTypes))
		for _, ct := range s.httpCfg.BodyAllowedContentTypes {
			s.allowedContentSet[strings.ToLower(strings.TrimSpace(ct))] = struct{}{}
		}

		if s.scrubCfg.Enabled {
			for _, pattern := range s.scrubCfg.SensitivePatterns {
				if re, err := regexp.Compile(pattern); err == nil {
					s.compiledPatterns = append(s.compiledPatterns, re)
				}
			}
		}
	})
}

// ScrubHeaders filters and redacts HTTP headers. Sensitive headers are always
// redacted regardless of scrub config. Returns key-value pairs suitable for
// span attributes.
func (s *HTTPScrubber) ScrubHeaders(headers map[string][]string, allowed []string) map[string]string {
	result := make(map[string]string, len(headers))
	allowedSet := s.buildAllowedSet(allowed)

	for name, values := range headers {
		lower := strings.ToLower(name)

		// If an allow-list is configured, skip headers not in it
		if len(allowedSet) > 0 {
			if _, ok := allowedSet[lower]; !ok {
				continue
			}
		}

		value := strings.Join(values, ", ")

		// Always redact sensitive headers
		if _, sensitive := s.sensitiveHeaderSet[lower]; sensitive {
			value = s.redactedValue()
		}

		result[lower] = value
	}

	return result
}

// ScrubQueryString redacts sensitive query parameter values.
// Only scrubs when ScrubConfig.Enabled is true.
func (s *HTTPScrubber) ScrubQueryString(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}

	if !s.scrubCfg.Enabled {
		return rawQuery
	}

	// Parse and rebuild query string, redacting sensitive param keys
	parts := strings.Split(rawQuery, "&")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			result = append(result, part)
			continue
		}

		key := kv[0]
		if s.isKeyMatch(key) {
			result = append(result, key+"="+s.redactedValue())
		} else {
			result = append(result, part)
		}
	}

	return strings.Join(result, "&")
}

// ScrubBody truncates and redacts sensitive patterns in body content.
// Returns the scrubbed body string.
func (s *HTTPScrubber) ScrubBody(body string, maxSize int) string {
	if body == "" {
		return ""
	}

	// Truncate
	if maxSize > 0 && len(body) > maxSize {
		body = body[:maxSize] + "...[truncated]"
	}

	// Apply pattern-based redaction when scrubbing is enabled
	if s.scrubCfg.Enabled {
		for _, re := range s.compiledPatterns {
			body = re.ReplaceAllString(body, s.redactedValue())
		}
	}

	return body
}

// IsAllowedContentType checks if the content-type is eligible for body capture.
func (s *HTTPScrubber) IsAllowedContentType(contentType string) bool {
	if len(s.allowedContentSet) == 0 {
		return true
	}

	ct := strings.ToLower(strings.TrimSpace(contentType))
	// Strip parameters (e.g., "application/json; charset=utf-8" -> "application/json")
	if idx := strings.IndexByte(ct, ';'); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}

	_, ok := s.allowedContentSet[ct]
	return ok
}

func (s *HTTPScrubber) redactedValue() string {
	if s.scrubCfg.RedactedValue != "" {
		return s.scrubCfg.RedactedValue
	}
	return "[REDACTED]"
}

func (s *HTTPScrubber) buildAllowedSet(allowed []string) map[string]struct{} {
	if len(allowed) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(allowed))
	for _, h := range allowed {
		set[strings.ToLower(h)] = struct{}{}
	}
	return set
}

func (s *HTTPScrubber) isKeyMatch(key string) bool {
	lower := strings.ToLower(key)
	for _, re := range s.compiledPatterns {
		if re.MatchString(lower) {
			return true
		}
	}
	return false
}
