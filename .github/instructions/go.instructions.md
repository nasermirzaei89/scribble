---
applyTo: '**/*.go'
description: This file describes the Go code style for the project.
---

# Go Code Style Instructions

## Commands

For running go actions use the following commands:

### Install Dependencies

```bash
make go-mod-download
```

### Format Code

Run it to format Go files:

```bash
make go-format
```

### Lint Code

Golangci-lint checks the code for linting issues.
It should be run before committing code, or after making significant changes.

```bash
make go-lint
```

### Build Code

```bash
make go-build
```

### Test Code

```bash
make go-test
```

## Logging

Use `log/slog` for structured logging.

Be careful with using `log.Fatal` as it exits the application immediately and ignores deferred calls. Prefer using `panic` in situations where you want to ensure deferred calls are executed.

## Error Handling

Use `fmt.Errorf` with `%w` verb to wrap errors for better context.

Always check and handle errors returned from functions.

Logging an error with error level means you have handled it.
If you are just propagating the error, do not log it; let the caller handle it.
