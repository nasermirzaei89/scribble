# Agents Instructions

<https://agents.md>

## Project Overview

Scribble is a tiny public writing place where real humans can post simple thoughts.

## UI Theme

> TBD

## Tech Stack

- **Backend**: Go 1.26.0
- **Frontend**: TypeScript, HTMX, AlpineJS, Tailwind CSS
- **Templates**: Go HTML templates (`.gohtml`)

## Development Commands

**Before starting work:**

- `make go-mod-download` - Install dependencies

**During development:**

- `make go-format` - Format code (run before committing)
- `make go-lint` - Check code quality
- `make go-run` - Run the application locally

**Before committing:**

- `make` - Run full validation (format, lint, build, test)
- `make go-test` - Run tests only
- `make go-build` - Build binary only

**Default workflow:** Run `make` to execute format → lint → build → test sequence.

## Code Structure

- `/cmd/scribble/` - Application entry point
- `/web/templates/` - Go HTML templates
- `/web/assets/` - Source TypeScript and CSS
- `/web/static/` - Compiled frontend assets

## Guidelines

- It uses modular monolith architecture
- Keep Go code idiomatic and simple
- Use htmx for dynamic interactions
- Templates use `.gohtml` extension
- Frontend builds to `/web/static/`
- Follow existing patterns in codebase

## Code style

For code style refer to [GitHub Instructions](.github/instructions/).

## Commit Message Guidelines

When creating commits, follow these conventions:

### Commit Title

- Use natural language that completes the phrase: **"[This commit will] ..."**
- Start with a capital letter
- Use imperative mood (e.g., "Add", "Fix", "Update", "Remove")
- Keep it concise (50-72 characters recommended)
- Do NOT use conventional commit prefixes like `feat:`, `fix:`, `test:`, `chore:`, etc.

**Good examples:**

- "Add comprehensive repository tests for SQLite implementations"
- "Refactor database layer to use direct sql.DB instances"
- "Update email field naming to emailAddress across codebase"

**Bad examples:**

- "test: add tests" (uses prefix)
- "added some stuff" (not imperative, too vague)
- "Fix bug" (too vague, no context)

### Commit Description

- Include a blank line between title and description
- Use bullet points to describe what changed
- Be specific about the changes made
- Group related changes together
- Focus on WHAT changed and WHY, not HOW

**Example:**

```txt
Add comprehensive repository tests for SQLite implementations

- Add OTP repository tests covering CRUD operations, validation, and edge cases
- Add session repository tests for user sessions and expiration handling
- Add user repository tests for all user operations and status verification
- Tests cover multiple scenarios including resends, multiple sessions, and cleanup operations
```
