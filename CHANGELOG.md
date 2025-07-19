# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of go-common library
- `syscmd` package for system command execution with fluent API
- Support for command timeouts, retries, and context cancellation
- Cross-platform compatibility (Windows, macOS, Linux)
- Comprehensive test coverage
- GitHub Actions CI/CD pipeline
- Automated security scanning with Gosec
- Code quality checks with golangci-lint

### Features
- Fluent interface for command building
- Exponential backoff retry mechanism
- Context-aware command execution
- Predefined command configurations (Quick, Resilient)
- Silent execution mode
- Built-in error handling and logging 
