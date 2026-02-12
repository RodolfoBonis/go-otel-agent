# Contributing to go-otel-agent

Thank you for your interest in contributing! This guide will help you get started.

## Development Setup

### Prerequisites

- Go 1.24+
- [golangci-lint](https://golangci-lint.run/welcome/install/)
- A running OpenTelemetry collector (optional, for integration testing)

### Getting Started

```bash
# Clone the repository
git clone https://github.com/RodolfoBonis/go-otel-agent.git
cd go-otel-agent

# Download dependencies
go mod download

# Run tests
go test ./...

# Run linter
golangci-lint run

# Build
go build ./...
```

## How to Contribute

### Reporting Bugs

- Use the [Bug Report](https://github.com/RodolfoBonis/go-otel-agent/issues/new?template=bug_report.yml) issue template
- Include your Go version, library version, and observability backend
- Provide a minimal reproduction if possible

### Suggesting Features

- Use the [Feature Request](https://github.com/RodolfoBonis/go-otel-agent/issues/new?template=feature_request.yml) issue template
- Describe the problem you're solving and your proposed API

### Submitting Pull Requests

1. **Fork** the repository and create your branch from `main`
2. **Follow** the existing code style and conventions
3. **Add tests** for any new functionality
4. **Ensure** all tests pass: `go test -race ./...`
5. **Ensure** linting passes: `golangci-lint run`
6. **Write** a clear PR description using the template

### Branch Naming Convention

- `feat/description` — New features
- `fix/description` — Bug fixes
- `perf/description` — Performance improvements
- `refactor/description` — Code refactoring
- `docs/description` — Documentation updates
- `ci/description` — CI/CD changes
- `test/description` — Test additions or fixes

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add per-route sampling configuration
fix: resolve nil meter panic when metrics disabled
perf: cache metric instruments in sync.Map
docs: add GORM plugin usage example
test: add scrub processor edge case tests
ci: increase golangci-lint timeout
refactor: extract config types to separate package
```

## Code Standards

### General

- Follow idiomatic Go conventions
- Use `go fmt` and `goimports` for formatting
- All exported types and functions must have doc comments
- No unused imports or variables
- Handle all errors (no `_` for error returns in production code)

### Testing

- Use table-driven tests where appropriate
- Test both positive and negative paths
- Use `t.Setenv()` for environment variable tests
- Clean up resources with `defer` (e.g., `agent.Shutdown()`)
- Use `testing.Short()` to skip long-running tests

### Performance

- Cache metric instruments (use `sync.Map`)
- Avoid allocations in hot paths
- Use bounded cardinality for metric attributes
- Profile before optimizing

### Security

- Never log sensitive data (tokens, passwords, PII)
- Use the PII scrubbing feature for span attributes
- Validate all external input
- Keep dependencies up to date

## Project Structure

```
go-otel-agent/
├── agent.go            # Core agent lifecycle
├── config.go           # Configuration and env var loading
├── config/types.go     # Config type definitions
├── options.go          # Functional options
├── errors.go           # Sentinel errors
├── health.go           # Health probes
├── noop.go             # Noop implementations
├── logger/             # Structured logging
├── provider/           # OTel providers (trace, metric, log)
├── helper/             # Tracing and metrics helpers
├── collector/          # Metric collectors (runtime, business, etc.)
├── instrumentor/       # Auto-instrumentation
├── internal/matcher/   # Route exclusion matcher
├── integration/        # Framework integrations (Gin, GORM, Redis, AMQP)
├── fxmodule/           # Uber FX module
└── examples/           # Usage examples
```

## Release Process

Releases are automated via GitHub Actions:

1. All changes go through PRs to `main`
2. PRs require passing CI (lint + tests)
3. Maintainers tag releases with semantic versioning: `v0.1.0`, `v0.2.0`, `v1.0.0`
4. Tags trigger the release workflow which creates a GitHub release and updates the Go module proxy

## Getting Help

- Open a [Discussion](https://github.com/RodolfoBonis/go-otel-agent/discussions) for questions
- Check existing [Issues](https://github.com/RodolfoBonis/go-otel-agent/issues) before creating new ones
- Review the [README](README.md) for usage documentation

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
