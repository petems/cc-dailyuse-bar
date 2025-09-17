# Repository Guidelines

## Project Structure & Module Organization
- `src/main.go` bootstraps the tray app and wires services.
- `src/services/` manages ccusage polling and configuration access; `src/models/` holds alert, config, and usage types; shared helpers live in `src/lib/`.
- Tests are grouped under `tests/` (`unit`, `integration`, `contract`); assets and docs sit in `docs/`.
- Systemd packaging lives in `cc-dailyuse-bar.service`; Go modules are tracked by `go.mod`/`go.sum`.

## Build, Test, and Development Commands
- `make build` produces the tray binary in the workspace; `make build-linux` targets Linux cross-builds.
- `make run` runs the app via `go run ./src`; add `--daemon` to stay in the background.
- `make test`, `make test-race`, and `make bench` run unit tests, race tests, and benchmarks respectively.
- `make coverage` emits `coverage.out` and an HTML report; `make security` checks dependencies with `nancy`.
- `make lint`, `make lint-fix`, and `make fmt` apply `golangci-lint` and `gofmt` rules; `make check` chains lint, tests, and build.

## Coding Style & Naming Conventions
- Follow Go defaults: tabs for indentation, `gofmt` for layout, and idiomatic receiver naming (`cfg`, `svc`, etc.).
- Exported types/functions live in `UpperCamelCase`; locals stay `lowerCamelCase`.
- Favor structured logging helpers in `src/lib/logger.go`; errors should wrap context using the utilities in `src/lib/errors.go`.
- Run `make format` before pushing to auto-fix lint and formatting issues.

## Testing Guidelines
- Write fast unit tests in `tests/unit`, scenario coverage in `tests/integration`, and CLI/API contract checks in `tests/contract`.
- Use Go’s `testing` package with `testify` assertions; name files `*_test.go` and keep table tests for variations.
- Ensure new behavior maintains or improves coverage (`make coverage-func` highlights gaps).

## Commit & Pull Request Guidelines
- Use Conventional Commit prefixes (e.g., `feat:`, `fix:`, `ci:`) as seen in recent history.
- Commits should bundle a logical change set and include tests or docs when applicable.
- Pull requests need a summary of changes, test evidence (`make test` output), and linked issues or screenshots for UI updates.
- Request review once CI is green; note any follow-up work explicitly in the PR description.

## Security & Configuration Tips
- Keep `ccusage` accessible in PATH; document overrides via `config.yaml` in the XDG config dir.
- Validate YAML changes with `go test ./src/models/...` when adjusting config structures.
- Run `make security` before releases to surface vulnerable dependencies.
