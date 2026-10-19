# CLAUDE.md

Go service that stores and serves transactions, categories, subcategories and locations for the personal finance app. Callers: the BFF (used by financer) and mcp-api. Solo project.

## Commands
- `make verify` - every check (gofmt, vet, build, tests, golangci-lint). Use this to verify.
- `make build` / `make run` / `make test` / `make lint`
- `go test ./internal/service -run TestName -count=1` - one test
- Integration tests in `internal/repository` need `TEST_DATABASE_DSN` (real Postgres); they skip without it.

## Architecture
- `cmd/server/main.go` wires everything: config, DB, migrations, repos, services, handlers, Gin routes under `/api/v2`.
- Layers: `handler` (HTTP, validation) -> `service` (business rules) -> `repository` (GORM/SQL). Handlers never query the DB (they only map GORM errors to HTTP codes).
- `internal/migrations/*.sql` are embedded and applied once each, in filename order, at boot (`RunMigrations`). AutoMigrate is not used.
- `service/identity_client.go` and `service/insights_notifier.go` call other services over HTTP; both are optional (empty URL disables them).
- Config comes only from env vars (`internal/config/config.go`); `stack.env` holds real values and is never read by Claude.

## Conventions
- Service tests use hand-written repository mocks in the same `_test.go` file; follow that pattern, no mocking library.
- Money is `decimal(12,2)` in Postgres and `float64` in Go; round at the edges, never compare floats with `==` in new logic.
- Report periods use the reporting timezone (`cfg.Reporting`), not UTC.
- Rows are soft-deleted (`deleted_at`); queries keep the GORM soft-delete scope unless they deliberately include deleted rows (see `categoryRepository.FindAll`).

## Things Claude gets wrong
- Editing an existing migration. Applied files never re-run; always add a new `internal/migrations/YYYYMMDD_name.sql`. Do not use `make migrate-create` (it writes to `db/migrations/`, which is not embedded).
- Trusting `make lint`: it prints a message and exits 0 when golangci-lint is missing. `make verify` fails instead.
- Running `make fmt`: it also runs `go mod tidy` and can change `go.mod`/`go.sum`. Format a single file with `gofmt -s -w <file>` (the edit hook already does).
- Using the deprecated `Transaction.Subtype` field in new code.
- Changing a JSON field name or route without checking the BFF and mcp-api, which depend on it.

## Verifying your work
Before saying a task is done, run `make verify` and paste the relevant output (at least the summary line and any failure). If something fails, say so; do not call the task done.
Known baseline failures (not caused by your change, fix only if the task is about them): gofmt on 3 files (`internal/model/transactions.go`, `internal/repository/transactions_repository.go`, `internal/service/transactions_service.go`) and 5 golangci-lint issues (3 errcheck on `resp.Body.Close`, 2 staticcheck). Any failure beyond these is yours.

## Process
Non-trivial changes follow `docs/sdlc/README.md`: `/sdlc-intent` -> `/sdlc-spec` -> `/sdlc-plan` -> implement -> `/sdlc-review` -> commit. Review rules: `REVIEW.md`.
