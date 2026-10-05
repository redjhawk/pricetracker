# Commit step: Amazon instalment price (issue #45)

- Gates: specifications ready, API contract unchanged (backend-only), review has no findings, decisions record no critical blocker.
- Staged scope: `internal/amazon/collector.go`, `internal/amazon/collector_test.go`, `doc/specifications/amazon-price-detection/`, `doc/changes/amazon-installment-price/`.
- Checks: `go test ./...` passes; `go vet ./internal/amazon/` clean; `git diff --cached --check` clean.
- Size: about 315 changed lines, so one PR (below the 450±50 target).
- Commits:
  1. `docs(amazon): specify instalment-proof price detection`
  2. `fix(amazon): ignore instalment amounts when detecting price`
- Branch: `ai-dev/issue-45-20261005-2049`, pushed with `git push -u origin HEAD`.
- PR: `ai-dev #45: ignore Amazon instalment amounts in price detection`, base `master`, closes #45.
