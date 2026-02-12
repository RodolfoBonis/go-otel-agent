package matcher

import (
	"path"
	"strings"
)

// RouteMatcher determines if a route should be excluded from instrumentation.
// It is pre-compiled at construction time for performance.
type RouteMatcher struct {
	exactPaths  map[string]struct{}
	prefixPaths []string
	patterns    []string
}

// RouteExclusionConfig configures which routes to exclude.
type RouteExclusionConfig struct {
	ExactPaths  []string // O(1) map lookup: ["/health", "/metrics"]
	PrefixPaths []string // strings.HasPrefix: ["/debug/", "/internal/"]
	Patterns    []string // path.Match glob: ["/api/v*/health"]
}

// NewRouteMatcher creates a pre-compiled route matcher.
func NewRouteMatcher(cfg RouteExclusionConfig) *RouteMatcher {
	exact := make(map[string]struct{}, len(cfg.ExactPaths))
	for _, p := range cfg.ExactPaths {
		exact[p] = struct{}{}
	}

	// Normalize prefixes - ensure they end with /
	prefixes := make([]string, 0, len(cfg.PrefixPaths))
	for _, p := range cfg.PrefixPaths {
		if p != "" {
			prefixes = append(prefixes, p)
		}
	}

	patterns := make([]string, 0, len(cfg.Patterns))
	for _, p := range cfg.Patterns {
		if p != "" {
			patterns = append(patterns, p)
		}
	}

	return &RouteMatcher{
		exactPaths:  exact,
		prefixPaths: prefixes,
		patterns:    patterns,
	}
}

// ShouldExclude returns true if the given path should be excluded.
func (m *RouteMatcher) ShouldExclude(requestPath string) bool {
	if m == nil {
		return false
	}

	// Layer 1: exact match (O(1))
	if _, ok := m.exactPaths[requestPath]; ok {
		return true
	}

	// Layer 2: prefix match
	for _, prefix := range m.prefixPaths {
		if strings.HasPrefix(requestPath, prefix) {
			return true
		}
	}

	// Layer 3: glob pattern match
	for _, pattern := range m.patterns {
		if matched, _ := path.Match(pattern, requestPath); matched {
			return true
		}
	}

	return false
}

// IsEmpty returns true if no exclusions are configured.
func (m *RouteMatcher) IsEmpty() bool {
	if m == nil {
		return true
	}
	return len(m.exactPaths) == 0 && len(m.prefixPaths) == 0 && len(m.patterns) == 0
}
