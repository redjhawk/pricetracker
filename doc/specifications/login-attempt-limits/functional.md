# Functional specification: login attempt limits by client address

Status: ready (issue #266); FR-LOGIN-LIM-012–014 implemented (issue #284, maintenance: no behaviour change)
Owner: functional specification agent
User decision/reference: GitHub issue #266 ("Login: limit failed attempts by client address instead of locking the username").

## Purpose and scope

Today failed logins are counted per username: after 5 consecutive failures the username is refused for 1 minute for everyone ([FR-AUTH-005](../multi-user-login/functional.md)), so anyone typing wrong passwords for a username keeps its real user, or the administrator, out. Password checks also run one at a time with no bound on how many wait, so a flood of attempts makes every other login wait. The user asks that only the failing client be refused, that refused clients cost no password check, and that logins be answered at once when too many password checks are already waiting.

The application is reachable only through Cloudflare, which sends the client's address in the `CF-Connecting-IP` request header.

Actors: a person logging in (regular user or administrator) and any other client sending login attempts. Cloudflare is the external system in front of the application.

Included: how failed logins are counted and refused; the answer to a login when too many password checks are waiting.

Excluded: the numbers 5 failures and 1 minute, the generic wrong-credentials error, sessions, logout, open and protected modes ([multi-user-login](../multi-user-login/functional.md)), which stay unchanged; any limit on failures by one client across different usernames; grouping of neighbouring addresses (each address is a client of its own); administrator user creation and the `admin-password` device command, which keep waiting for their password computation as today; any change to the login screen other than the message it shows.

Terms:

- **Client address:** the value of the `CF-Connecting-IP` header when the request carries one that is a valid IPv4 or IPv6 address; otherwise the address of the network connection the request came from.
- **Pair:** a client address together with a username, the username compared case-insensitively (`Alice` and `alice` are the same username).
- **Password check:** the verification of a submitted password, which takes about one second on the target device and runs one at a time.
- **Locked pair:** a pair refused for 1 minute after 5 consecutive failed logins.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-LOGIN-LIM-001 | A login fails (wrong password, or a username that does not exist). | The failure is counted for its pair only. Failures of one pair never count toward another pair. Unknown usernames are counted the same way as existing ones. | From address A, 4 wrong passwords for `alice`, then 1 wrong password from address B for `alice`: the next attempt from A with the right password logs in. |
| FR-LOGIN-LIM-002 | A pair reaches 5 consecutive failed logins. | Further attempts of that pair are refused for 1 minute after the 5th failure, even with the right password, with the message "Too many failed attempts. Wait one minute and try again." (HTTP 429). | From address A, 5 wrong passwords for `alice`; a 6th attempt from A for `alice` within the minute, with the right password, is refused with that message. |
| FR-LOGIN-LIM-003 | A pair is locked. | Every other pair is unaffected: the same username from another client address, and other usernames from the same client address, log in normally. | While A/`alice` is locked, `alice` logs in from address B with the right password, and `bob` logs in from address A with the right password. |
| FR-LOGIN-LIM-004 | An attempt arrives for a locked pair. | It is refused at once without a password check, so it does not wait behind other password checks and does not delay them. It is not counted as a failure and does not extend the lock. | While A/`alice` is locked and another login's password check is running, an attempt from A for `alice` is answered 429 before that check ends; 1 minute after the 5th failure, A logs in as `alice` with the right password. |
| FR-LOGIN-LIM-005 | A pair logs in successfully. | Its count of consecutive failures restarts at zero. Other pairs' counts are unchanged. | From A, 4 wrong passwords for `alice`, a success, then 4 wrong passwords: the next attempt from A with the right password logs in. |
| FR-LOGIN-LIM-006 | A lock ends, or a pair has had no failure for 1 hour. | The pair's failures are forgotten: after a lock ends, 5 new consecutive failures are needed to lock it again; failures older than 1 hour without a later failure no longer count. Failures are kept only for pairs with a recent failure or an active lock, so they cannot accumulate without bound. | After a lock of A/`alice` ends, 4 wrong passwords from A then the right one log in. From A, 4 wrong passwords for `alice`, then 1 hour and a moment later 4 more wrong passwords: the next attempt with the right password logs in. |
| FR-LOGIN-LIM-007 | A login request carries a `CF-Connecting-IP` header with a valid IPv4 or IPv6 address. | That address is the client address, whatever the connection address. Requests coming through the same connection or proxy with different header addresses are different clients. | Two requests from the same connection, with `CF-Connecting-IP` 203.0.113.1 and 203.0.113.2: after 5 failures for `alice` with the first address, `alice` logs in with the second address. |
| FR-LOGIN-LIM-008 | A login request has no `CF-Connecting-IP` header, or its value is empty or not a valid IP address. | The connection's address is the client address. | Without the header, 5 failures for `alice` from one connection address lock `alice` for that address only; a request whose header reads `not-an-ip` from the same connection address is refused as locked. |
| FR-LOGIN-LIM-009 | A login needs a password check while the fixed maximum number of password checks are already waiting to run (the maximum is set by the technical specification and is at least 1). | The login is answered at once, without a password check and without waiting, with HTTP 429 and the message "Too many login attempts are being checked. Try again in a moment." It is not counted as a failure for its pair. | With password checks held running and the maximum number waiting, a further login with the right password is answered 429 with that message without waiting; once the waiting checks have finished, the same login succeeds. |
| FR-LOGIN-LIM-010 | Fewer than the maximum number of password checks are waiting. | A login waits its turn and is then checked, as today; logins are never refused as busy while a slot to wait is free. | With one password check running and none waiting, a second login with the right password succeeds after the first check ends. |
| FR-LOGIN-LIM-011 | A refused login (locked pair or too many waiting checks) is shown on the login screen. | The login screen shows the server's message in its existing "Could not log in" error, and the person stays logged out. | After a 429 for a locked pair or for too many waiting checks, the login screen shows the corresponding message and no session is created. |

### Specification index status (issue #284, maintenance)

PR #283 delivered FR-LOGIN-LIM-001 to 011 (issue #266), but the Login attempt limits row of the specification index still says they are only specified. The index must say they are implemented. No behaviour change.

- **FR-LOGIN-LIM-012** The status cell of the Login attempt limits row in the Subject specifications table of `doc/FUNCTIONAL_SPECIFICATIONS.md` begins with “implemented (issue #266)”, as the other rows of delivered subjects say “implemented”. Check: that row contains “implemented (issue #266)” and does not contain “specified (issue #266)”.
- **FR-LOGIN-LIM-013** Only that row of the index changes. The other rows, including the other rows that still say “specified (issue #NNN)” and the Multiple users and login row, are unchanged; updating them is out of scope of issue #284. Check: in the change's diff of `doc/FUNCTIONAL_SPECIFICATIONS.md`, the only changed line is the Login attempt limits row.
- **FR-LOGIN-LIM-014** Nothing else changes: the text of FR-LOGIN-LIM-001 to 011, `technical.md` of this subject, the change records under `doc/changes/` of issue #266 and the files under `doc/todo/` are unchanged; the change's diff outside `doc/` is empty; `go vet ./...` and `go test ./...` pass, and the user observes no change in the application.

## States and corner cases

- Wrong-credential answers, the 5-failure threshold and the 1-minute lock duration are unchanged; only who is counted and refused changes.
- An attempt that was already waiting for its password check when its pair became locked is still checked; if it fails, the lock stays as it is (not extended); if it succeeds, the person logs in and the pair's count restarts (as today).
- A login cancelled by the client while waiting for its password check is not counted.
- A busy refusal (FR-LOGIN-LIM-009) and a locked refusal (FR-LOGIN-LIM-002) are distinguished by their messages.
- The `CF-Connecting-IP` header is trusted because the application is reachable only through Cloudflare; installations that expose the application directly are out of scope.
- No new screen, setting or loading state; the login button behaves as today while a login is being checked.

## Acceptance criteria summary

- 5 failed logins from one client address for a username refuse only that client for that username, for 1 minute; the real user logs in from elsewhere at the same time.
- Refused attempts of a locked pair are answered at once and never compute a password check.
- When the maximum number of password checks are already waiting, a login is answered 429 at once with a busy message.
- The client address comes from `CF-Connecting-IP` when valid, else from the connection.
- Failure memory stays bounded: expired locks and failures older than 1 hour are forgotten.

## Supersedes

- [FR-AUTH-005](../multi-user-login/functional.md) and D-11: failed logins are counted per pair (client address and username), not per username.
