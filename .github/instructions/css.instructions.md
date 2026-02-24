---
applyTo: '**/*.css'
description: This file describes the CSS code style for the project.
---

# CSS Code Style Instructions

All CSS code in this project should follow Tailwind CSS conventions and best practices. This includes using utility classes for styling and avoiding custom CSS where possible.
Current Tailwind CSS version is in the [web/package.json](../../web/package.json) file.

## Commands

For running CSS actions use the following commands:

### Format Code

Run it to format CSS files:

```bash
make npm-format
```

### Lint Code

ESLint checks the code for linting issues.
It should be run before committing code, or after making significant changes.

```bash
make npm-lint
```

### Build Code

```bash
make npm-build
```

### Test Code

Not Available for CSS files.

## Logging

Use `log/slog` for structured logging.

Be careful with using `log.Fatal` as it exits the application immediately and ignores deferred calls. Prefer using `panic` in situations where you want to ensure deferred calls are executed.

## Error Handling

Use `fmt.Errorf` with `%w` verb to wrap errors for better context.

Always check and handle errors returned from functions.

Logging an error with error level means you have handled it.
If you are just propagating the error, do not log it; let the caller handle it.

## Google Style Guide

Follow the [Google HTML/CSS Style Guide](https://google.github.io/styleguide/htmlcssguide.html) for more best practices and conventions in HTML and CSS programming.

## Components

All defined components should have a prefix of `as-` to avoid naming conflicts and to clearly indicate that they are part of the project.
For example, a button component should be named `as-button` instead of just `button`.

Boolean attributes should be prefixed with `is-` to indicate their boolean nature. For example, an active state could be represented as `is-active`.

Other attributes should be prefixed with the attribute name followed by a hyphen. For example, a variant attribute could be represented as `variant-outlined` or `variant-text`.
