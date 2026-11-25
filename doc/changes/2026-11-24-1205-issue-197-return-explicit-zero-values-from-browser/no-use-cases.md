# No use cases for issue #197

This change is code maintenance with no user-visible behaviour, so it has no use cases.

It changes only the return values of `profile()` in `internal/amazon/browser.go` when the persistent profile's stale lock cannot be discarded (see `doc/specifications/amazon-browser-profile/functional.md`). The requirements cover only internal return values and the build:

- **FR-AMZ-BPROF-008**: on that failure, `profile()` returns an empty path, a `nil` cleanup and the error, like its branch where the profile directory cannot be created.
- **FR-AMZ-BPROF-009**: on success with the persistent profile, `profile()` still returns the profile path, a cleanup that does nothing and no error.
- **FR-AMZ-BPROF-010**: the diff outside `doc/` is limited to `profile()` and its tests, `go vet ./...` and `go test ./...` pass, and the operator sees no change in the application.

A lock that cannot be discarded is still a counted "browser could not start" failure (FR-AMZ-BPROF-006), as in the existing use cases UC-AMZ-BPROF-001 and UC-AMZ-BPROF-002, which stay unchanged. No API, database, user interface or runtime behaviour changes, and no new actor goal or flow is involved. The test cases and the deterministic gates check these requirements directly.
