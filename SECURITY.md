# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in go-otel-agent, please report it responsibly.

### How to Report

1. **Do NOT** open a public issue for security vulnerabilities
2. Email the maintainer directly or use [GitHub's private vulnerability reporting](https://github.com/RodolfoBonis/go-otel-agent/security/advisories/new)
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

### What to Expect

- **Acknowledgment** within 48 hours
- **Assessment** within 1 week
- **Fix and disclosure** coordinated with reporter

### Security Best Practices

When using go-otel-agent:

- **Enable PII scrubbing** (`OTEL_PII_SCRUB_ENABLED=true`) for production environments
- **Use TLS** for connections to external collectors
- **Rotate** SigNoz access tokens periodically
- **Review** span attributes to avoid leaking sensitive data
- **Keep** the library updated to the latest version
- **Never** commit tokens or credentials in configuration
- **Keep body capture disabled** in production unless specifically needed (`OTEL_HTTP_CAPTURE_REQUEST_BODY` and `OTEL_HTTP_CAPTURE_RESPONSE_BODY` default to `false`)
- **Verify sensitive headers list** covers your custom auth headers (`OTEL_HTTP_SENSITIVE_HEADERS`)
- **Limit body content types** to text-based formats only (`OTEL_HTTP_BODY_ALLOWED_CONTENT_TYPES`)

### Automated Security Scanning

This project runs automated security checks:

- **govulncheck** — Weekly vulnerability scanning of Go dependencies
- **go mod verify** — Dependency integrity verification on every PR
- **Dependabot** — Automated dependency update PRs
