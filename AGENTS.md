# Repository Guidelines

## Project Structure & Modules
- `cmd/httpserver/`: CLI entrypoint and service startup.
- `httpserver/`: HTTP server, handlers, routing, and tests.
- `application/`, `domain/`, `ports/`, `adapters/`: Clean/hexagonal layers (core logic, interfaces, DB/secrets adapters).
- `schema/`: SQL migrations; `testdata/`: fixtures; `metrics/`: Prometheus metrics; `docker/`: Dockerfiles and compose; `scripts/`: CI/e2e helpers.

## Build, Test, and Dev Commands
- `make build`: Build the server to `build/builder-hub`.
- `go run cmd/httpserver/main.go`: Run locally (see env flags below).
- `make test`: Run unit tests. Use `RUN_DB_TESTS=1 make test` to include DB/e2e tests.
- `make cover` / `make cover-html`: Coverage summary / HTML report.
- `make fmt`: Format and tidy imports/modules.
- `make lint`: `go vet`, `staticcheck`, `golangci-lint`, and format checks.
- `make docker-httpserver`: Build the Docker image.
- Helpful: `docs/devenv-setup.md`, `scripts/ci/integration-test.sh`.

## Coding Style & Conventions
- Go formatting is enforced: run `make fmt` (gofmt, gofumpt, gci, go mod tidy).
- Always run `make fmt` and `make lint`.
- If touching `httpserver/e2e_test.go`, always test with database enabled: `make dev-postgres-restart test-with-db dev-postgres-stop`.
- Package names: lower-case; exported symbols: PascalCase; locals: camelCase.
- Keep files focused; group handlers in `httpserver`, business logic in `application/domain`.
- Imports: standard → third-party → local (enforced by gci).
- Indentation: Use tab, not spaces

## Testing Guidelines
- Framework: Go `testing`; tests live in `*_test.go`. Use `testdata/` fixtures.
- Quick run: `make test`; with DB/e2e: `RUN_DB_TESTS=1 make test`.
- Coverage: `make cover`. Prefer table-driven tests and clear Arrange/Act/Assert sections.

## Commit & Pull Request Guidelines
- Commits: concise, imperative mood (e.g., "Add API spec tests"); optional scope tags (e.g., `[CI]`); reference issues/PRs when relevant.
- PRs: include description, linked issues, testing steps (curl examples for API changes), and screenshots/log snippets when useful.
- Required before merge: `make lt` (lint + tests) green, updated docs in `docs/` when API/behavior changes.

## Configuration & Security
- Common env flags (via CLI/env): `LISTEN_ADDR`, `ADMIN_ADDR`, `INTERNAL_ADDR`, `METRICS_ADDR`, `POSTGRES_DSN`, `LOG_JSON`, `LOG_DEBUG`, `PPROF`, `MOCK_SECRETS`, `AWS_BUILDER_CONFIGS_SECRET_NAME`/`_PREFIX`.
- Default DSN targets local Postgres; do not commit secrets. Use `MOCK_SECRETS=1` for local dev.
