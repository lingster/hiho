# Agent Guidelines

Welcome! Use these conventions throughout this repository:

- Prefer small, composable packages. Keep files under 300 lines.
- Follow SOLID and DRY principles. Isolate side effects behind interfaces for easy testing.
- Favor table-driven tests and test-driven development.
- Keep CLI/TUI user flows discoverable: document new commands in the README.
- Go style: idiomatic naming, clear errors, no blanket `panic`, and avoid try/catch around imports (not applicable in Go).
- Preserve tmux hygiene in tests—name sessions uniquely and clean them up.

Scope: entire repository (until another AGENTS.md overrides).
