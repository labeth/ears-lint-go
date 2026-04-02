# Contributing

Thanks for contributing to `ears-lint-go`.

## Development Setup

1. Install Go (1.22+).
2. Clone the repository.
3. Run tests:

```bash
go test ./...
```

## Code Guidelines

- Keep behavior deterministic.
- Keep scope limited to standalone EARS linting.
- Avoid fuzzy matching and external calls.
- Prefer small, focused pull requests.

## Pull Requests

Before opening a PR:

1. Run formatting and tests:

```bash
gofmt -w .
go test ./...
```

2. Update tests for behavior changes.
3. Update README if API behavior changed.

## Reporting Issues

Please include:

- input text (or minimal reproducer)
- catalog snippet used
- expected behavior
- actual behavior and diagnostics
