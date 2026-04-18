# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.2.0] - 2026-03-04

### Changed

- **BREAKING**: Removed progress bar UI implementation and tests
- **BREAKING**: Removed embedded HTTP runtime control panel entry (retained pure runtime read/write capabilities)
- **BREAKING**: Removed legacy plugin management paths and redundant adapter layers
- **BREAKING**: Removed legacy `converter` compatibility paths, unified to `Codec` architecture

### Fixed

- Fixed `DesensitizeText` fallback behavior when manager doesn't modify result
- Improved `DesensitizeSpecificType` fallback behavior
- Bank card/IP/URL/Chinese name desensitization now works correctly

### Added

- New `SubscribeWithOptions` with backpressure policies:
  - `drop_oldest` (default): discard oldest messages first
  - `drop_newest`: discard newest messages first
  - `block_with_timeout`: block with timeout
- New subscription statistics APIs:
  - `GetSubscriptionStats()`
  - `ListSubscriberStats()`
  - `GetSubscriberStats(id int64)`
- Main logging path remains non-blocking under high subscription pressure

### Removed

- Removed `internal/common/pool_compat` compatibility layer
- Removed legacy compatibility comments and historical paths

### Quality

- All tests pass: `go test ./...`
- Race detection passes: `go test -race ./...`
- Go vet passes: `go vet ./...`

## [v0.1.0] - 2024-01-15

### Added

- Initial release
- Multi-level logging: Trace, Debug, Info, Warn, Error, Fatal
- Dual format output: Text and JSON
- Colored terminal output with TTY detection
- Data Loss Prevention (DLP) system
- Modular architecture with Formatter/Middleware/Handler/Sink plugins
- Performance optimizations: tiered buffer pools, LRU cache, atomic operations
- Runtime control for dynamic configuration
- Log subscription mechanism
- File logging with rotation and compression

---

## Version Policy

This project follows [Semantic Versioning](https://semver.org/):

- **MAJOR**: Incompatible API changes
- **MINOR**: Backwards-compatible new features
- **PATCH**: Backwards-compatible bug fixes

### Go Version Support

| slog Version | Minimum Go Version |
| ------------ | ------------------ |
| v0.2.x       | Go 1.23            |
| v0.1.x       | Go 1.21            |

### Upgrade Guide

When upgrading from v0.1.x to v0.2.0:

1. Remove any HTTP runtime panel code (now removed)
2. Remove progress bar usage (now removed)
3. Update any legacy converter imports to Codec
4. Run tests to verify compatibility

[unreleased]: https://github.com/darkit/slog/compare/v0.2.0...HEAD
[v0.2.0]: https://github.com/darkit/slog/releases/tag/v0.2.0
[v0.1.0]: https://github.com/darkit/slog/releases/tag/v0.1.0
