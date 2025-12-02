# Contributing to go-tartuffe

Thank you for your interest in contributing to go-tartuffe!

## Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/go-tartuffe.git
   cd go-tartuffe
   ```
3. Create a branch for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Setup

Ensure you have Go 1.25+ installed, then:

```bash
# Download dependencies
go mod download

# Build
make build

# Run tests
make test

# Run linter
make lint
```

## Making Changes

### Code Style

- Run `go fmt ./...` before committing
- Run `go vet ./...` to check for issues
- Follow standard Go conventions and idioms

### Testing

- Add tests for new functionality
- Ensure existing tests pass: `make test`
- Integration tests are in `test/integration/`

### Commit Messages

- Use clear, descriptive commit messages
- Start with a verb (Add, Fix, Update, Remove, etc.)
- Reference issues when applicable: `Fix #123: description`

## Pull Request Process

1. Update documentation if needed
2. Ensure CI passes (tests, linting, build)
3. Request review from maintainers
4. Address feedback promptly

## Reporting Issues

When reporting bugs, please include:

- Go version (`go version`)
- Operating system
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs or error messages

## Feature Requests

Feature requests are welcome! Please:

- Check existing issues first
- Describe the use case
- Explain why existing features don't meet the need

## Code of Conduct

Be respectful and constructive in all interactions. We're all here to build something useful together.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
