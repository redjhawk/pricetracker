# Questions: issue #392

Context: the repository has no workflow that runs on pull requests. The factory runs its green gate (`go vet ./...`, `go test ./...`, the build and the e2e suite) on the `arc-dev` machine, which has no C compiler, and then merges the pull request at once. `release.yml` runs on `arc-build`, and only after the merge to `master`. Changes under `.github/` and `scripts/factory/` are never merged by the factory on its own, so you will merge this change by hand.

1. **Where and when should the race run happen, and what should a data race stop?**
   - a) In the factory's green gate, on every factory change. A data race stops the change before its pull request is merged, as a failing `go test` does. This needs a C compiler in the `arc-dev` image (`deploy/arc/Dockerfile.dev`).
   - b) In a new GitHub workflow that runs on every pull request to `master` (and on every push to `master`), on a machine with a C compiler. A data race shows as a failed check on the pull request. It stops nothing by itself, because the factory merges without waiting for checks, unless you make the check required in the branch protection of `master`.
   - c) In `release.yml`, before the build. A data race stops the release of that `master` commit, but the change is already merged.
   - d) A combination of the above (say which).

2. **Which tests should the race run cover?**
   - a) Only the issue #218 capture tests (`go test -race -count=1 -run '^TestCaptureOutput_Issue218_' ./internal/amazon/amazontest/`), as the suggested work says.
   - b) Those, plus the other `-race` runs that the test-suite-hygiene design asks for (the `TestBrowserProfile` runs in `internal/amazon`). These need Chromium on the machine and take longer.
   - c) The whole Go suite (`go test -race ./...`). This needs Chromium, and the race detector makes the run several times slower than `go test ./...` (today the slowest packages, `internal/amazon`, `internal/httpapi` and `internal/service`, each take 2.5 to 4 minutes without `-race`).

3. **Must one manual run be recorded?** The issue says "until then, record the result of one manual run on a machine that has gcc". The factory cannot do such a run.
   - a) No. The automated race run replaces it.
   - b) Yes. You run the command on a machine with gcc and give its output in a comment, and the change records it in the issue #218 implementation record (`doc/changes/2026-11-28-0457-issue-218-share-the-log-stdout-capture-test-helper/implementation.md`, run 1).
