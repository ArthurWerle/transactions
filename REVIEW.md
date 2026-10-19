# REVIEW.md

How changes in this repo are reviewed, by `/sdlc-review`, by the reviewer agent in CI, or by hand. Solo project: the goal is catching real defects, not enforcing taste.

## Severity
- **Important**: blocks the merge. A bug with a concrete failure path, a security or privacy problem, a requirement from spec.md not met, behavior outside plan.md, new logic without a test, a verify failure introduced by the change, or the verify output not shown.
- **Nit**: does not block. Naming, readability, small simplifications, comments. At most 5 per review; pick the ones that matter most and drop the rest.

Every Important finding states the failure path (input or state -> wrong result). If you cannot describe one, it is a Nit or nothing. Pre-existing problems in untouched code are not findings; mention them in one line at the end at most.

## Pass 1: Bugs and correctness
- Logic does what the spec says on the edges: empty lists, zero and negative amounts, month boundaries, the reporting timezone, recurring vs one-off transactions, soft-deleted rows.
- Errors are checked and returned with context; no swallowed `err`, no `panic` in request paths.
- SQL and GORM: correct `WHERE` with soft delete, no N+1 in loops over transactions, transactions (DB) used when several writes must succeed together.
- Migrations: new file only, idempotent where possible, safe on existing data, consistent with the Go model.
- API contract: JSON field names, status codes and routes unchanged unless the spec says so (BFF and mcp-api depend on them).
- Tests: new behavior has a test that would fail without the change.

## Pass 2: Security and privacy
- No secrets, tokens or DSNs in code, logs, tests or docs.
- SQL built only with GORM parameters or `?` placeholders, never string concatenation of input.
- Input from requests validated before use (IDs, dates, amounts, enums).
- Logs do not include full transaction payloads or personal data beyond IDs.
- New outbound HTTP calls have timeouts and do not forward user input into URLs unescaped.

## Pass 3: Conformance with spec and plan
Only when the change has `docs/sdlc/<slug>/`; otherwise check against CLAUDE.md conventions only.
- Every requirement R1..Rn is implemented and covered by a test or a stated manual check.
- Every changed file is listed in plan.md's "Files that change". Unlisted files are Important unless trivially incidental (imports, a test helper), in which case they are Nits.
- The "Proof" items were run and their output was shown.
- Points of attention in spec.md were handled, not just acknowledged.
- CLAUDE.md "Things Claude gets wrong" items were not repeated.

## Output
Verdict (`ready` or `changes needed`), then Important, then Nits, then the requirement table. `ready` means zero Important findings.
