# No use cases for issue #293

This change is source-code maintenance with no user-visible behaviour, so it has no use cases.

It moves the two doc-comment lines of `clientAddress` in `internal/httpapi/auth.go` back directly above `func clientAddress` (see `doc/specifications/session-cookie-security/functional.md`). The requirements cover only source comments and the build:

- **FR-COOKIE-007**: the `clientAddress` comment lines stand directly above `func clientAddress` and are its doc comment.
- **FR-COOKIE-008**: the doc comment of `secureRequest` holds only its own lines, and one blank line separates `secureRequest` from the `clientAddress` doc comment.
- **FR-COOKIE-009**: outside `doc/`, the diff only moves those comment lines; no code line changes, FR-COOKIE-001–006 and the login attempt limits behave as before, the unit-test gate passes, and the operator sees no change in the application.

The session cookie and login limit behaviour does not change, and existing use cases already describe it:

- UC-COOKIE-001, UC-COOKIE-002, UC-COOKIE-003 (session cookie `Secure` attribute at login and logout; FR-COOKIE-001–006)
- UC-LOGIN-LIM-001, UC-LOGIN-LIM-002, UC-LOGIN-LIM-003 (login attempt limits by client address)

No API, database, user interface or runtime behaviour changes, and no actor goal or flow is involved. The test cases and the deterministic gates check these requirements directly.
