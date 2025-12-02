# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly:

1. **Do not** open a public GitHub issue
2. Email the maintainers directly or use GitHub's private vulnerability reporting
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

## Response Timeline

- **Acknowledgment**: Within 48 hours
- **Initial Assessment**: Within 7 days
- **Resolution Target**: Within 30 days for critical issues

## Security Considerations

### JavaScript Injection

The `--allowInjection` flag enables JavaScript execution in predicates and responses. This is disabled by default for security reasons. Only enable it in trusted environments.

### Network Exposure

By default, the server binds to all interfaces. Use `--localOnly` to restrict access to localhost, or use `--host` to bind to a specific interface.

### API Authentication

Use `--apikey` to require authentication for API access in production environments.

### IP Whitelisting

Use `--ipWhitelist` to restrict which IP addresses can access the API.

## Best Practices

1. Run with `--localOnly` when possible
2. Use `--apikey` in production
3. Avoid `--allowInjection` unless necessary
4. Run the container as non-root (default in Docker image)
5. Use network policies to restrict access
