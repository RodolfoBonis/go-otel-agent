package matcher_test

import (
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/internal/matcher"
)

// ---------------------------------------------------------------------------
// Exact path matching (O(1) map lookup)
// ---------------------------------------------------------------------------

func TestShouldExclude_ExactMatch(t *testing.T) {
	m := matcher.NewRouteMatcher(matcher.RouteExclusionConfig{
		ExactPaths: []string{"/health", "/metrics", "/ready"},
	})

	tests := []struct {
		path string
		want bool
	}{
		{"/health", true},
		{"/metrics", true},
		{"/ready", true},
		{"/healthz", false},
		{"/health/", false},
		{"/api/health", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			if got := m.ShouldExclude(tc.path); got != tc.want {
				t.Errorf("ShouldExclude(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Prefix path matching
// ---------------------------------------------------------------------------

func TestShouldExclude_PrefixMatch(t *testing.T) {
	m := matcher.NewRouteMatcher(matcher.RouteExclusionConfig{
		PrefixPaths: []string{"/debug/", "/internal/"},
	})

	tests := []struct {
		path string
		want bool
	}{
		{"/debug/pprof", true},
		{"/debug/vars", true},
		{"/debug/", true},
		{"/internal/status", true},
		{"/internal/", true},
		{"/api/debug", false},
		{"/debugx", false},
		{"/api/internal/foo", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			if got := m.ShouldExclude(tc.path); got != tc.want {
				t.Errorf("ShouldExclude(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Glob pattern matching
// ---------------------------------------------------------------------------

func TestShouldExclude_GlobPattern(t *testing.T) {
	m := matcher.NewRouteMatcher(matcher.RouteExclusionConfig{
		Patterns: []string{"/api/v*/health", "/static/*.js"},
	})

	tests := []struct {
		path string
		want bool
	}{
		{"/api/v1/health", true},
		{"/api/v2/health", true},
		{"/api/v10/health", true}, // '*' in path.Match matches any non-separator sequence within a segment
		{"/static/app.js", true},
		{"/static/vendor.js", true},
		{"/static/app.css", false},
		{"/api/health", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			if got := m.ShouldExclude(tc.path); got != tc.want {
				t.Errorf("ShouldExclude(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ShouldExclude returns false for non-matching paths
// ---------------------------------------------------------------------------

func TestShouldExclude_NoMatch(t *testing.T) {
	m := matcher.NewRouteMatcher(matcher.RouteExclusionConfig{
		ExactPaths:  []string{"/health"},
		PrefixPaths: []string{"/debug/"},
		Patterns:    []string{"/api/v*/health"},
	})

	nonMatching := []string{
		"/api/v1/users",
		"/dashboard",
		"/",
		"/api/v1/orders/123",
	}

	for _, path := range nonMatching {
		t.Run(path, func(t *testing.T) {
			if m.ShouldExclude(path) {
				t.Errorf("ShouldExclude(%q) = true, expected false", path)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Empty matcher (no exclusions configured)
// ---------------------------------------------------------------------------

func TestShouldExclude_EmptyMatcher(t *testing.T) {
	m := matcher.NewRouteMatcher(matcher.RouteExclusionConfig{})

	paths := []string{"/health", "/api/v1/users", "/debug/pprof", "/"}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			if m.ShouldExclude(path) {
				t.Errorf("empty matcher should never exclude, but excluded %q", path)
			}
		})
	}
}

func TestShouldExclude_NilMatcher(t *testing.T) {
	var m *matcher.RouteMatcher

	if m.ShouldExclude("/anything") {
		t.Error("nil matcher should return false")
	}
}

// ---------------------------------------------------------------------------
// Combined exact + prefix + glob
// ---------------------------------------------------------------------------

func TestShouldExclude_Combined(t *testing.T) {
	m := matcher.NewRouteMatcher(matcher.RouteExclusionConfig{
		ExactPaths:  []string{"/health", "/metrics"},
		PrefixPaths: []string{"/debug/", "/internal/"},
		Patterns:    []string{"/api/v*/health", "/static/*.js"},
	})

	tests := []struct {
		path string
		want bool
	}{
		// exact
		{"/health", true},
		{"/metrics", true},
		// prefix
		{"/debug/pprof", true},
		{"/internal/config", true},
		// glob
		{"/api/v1/health", true},
		{"/static/bundle.js", true},
		// none
		{"/api/v1/users", false},
		{"/login", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			if got := m.ShouldExclude(tc.path); got != tc.want {
				t.Errorf("ShouldExclude(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// IsEmpty
// ---------------------------------------------------------------------------

func TestIsEmpty_True(t *testing.T) {
	m := matcher.NewRouteMatcher(matcher.RouteExclusionConfig{})
	if !m.IsEmpty() {
		t.Error("expected IsEmpty() = true for empty config")
	}
}

func TestIsEmpty_NilMatcher(t *testing.T) {
	var m *matcher.RouteMatcher
	if !m.IsEmpty() {
		t.Error("expected IsEmpty() = true for nil matcher")
	}
}

func TestIsEmpty_False_WithExactPaths(t *testing.T) {
	m := matcher.NewRouteMatcher(matcher.RouteExclusionConfig{
		ExactPaths: []string{"/health"},
	})
	if m.IsEmpty() {
		t.Error("expected IsEmpty() = false with exact paths")
	}
}

func TestIsEmpty_False_WithPrefixes(t *testing.T) {
	m := matcher.NewRouteMatcher(matcher.RouteExclusionConfig{
		PrefixPaths: []string{"/debug/"},
	})
	if m.IsEmpty() {
		t.Error("expected IsEmpty() = false with prefix paths")
	}
}

func TestIsEmpty_False_WithPatterns(t *testing.T) {
	m := matcher.NewRouteMatcher(matcher.RouteExclusionConfig{
		Patterns: []string{"/api/v*/health"},
	})
	if m.IsEmpty() {
		t.Error("expected IsEmpty() = false with patterns")
	}
}

// ---------------------------------------------------------------------------
// Edge cases: empty strings in config are filtered out
// ---------------------------------------------------------------------------

func TestNewRouteMatcher_EmptyStringsFiltered(t *testing.T) {
	m := matcher.NewRouteMatcher(matcher.RouteExclusionConfig{
		PrefixPaths: []string{"", "/debug/", ""},
		Patterns:    []string{"", "/api/v*/health", ""},
	})

	if m.IsEmpty() {
		t.Error("expected non-empty matcher after filtering empty strings")
	}

	// The non-empty entries should still work.
	if !m.ShouldExclude("/debug/pprof") {
		t.Error("expected /debug/pprof to be excluded via prefix")
	}
	if !m.ShouldExclude("/api/v1/health") {
		t.Error("expected /api/v1/health to be excluded via pattern")
	}
}

// ---------------------------------------------------------------------------
// Prefix matching without trailing slash
// ---------------------------------------------------------------------------

func TestShouldExclude_PrefixWithoutTrailingSlash(t *testing.T) {
	m := matcher.NewRouteMatcher(matcher.RouteExclusionConfig{
		PrefixPaths: []string{"/internal"},
	})

	// /internal matches the prefix /internal
	if !m.ShouldExclude("/internal") {
		t.Error("expected /internal to be excluded")
	}
	// /internal/foo starts with /internal
	if !m.ShouldExclude("/internal/foo") {
		t.Error("expected /internal/foo to be excluded")
	}
	// /internalize also starts with /internal -- this is expected prefix behavior
	if !m.ShouldExclude("/internalize") {
		t.Error("expected /internalize to match prefix /internal")
	}
}
