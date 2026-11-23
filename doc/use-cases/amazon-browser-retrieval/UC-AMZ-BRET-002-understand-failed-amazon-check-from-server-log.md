# UC-AMZ-BRET-002 — Understand a failed Amazon check from the server log

**Primary actor:** Operator who installs and runs the server
**Supporting systems:** PriceFollower application, real web browser, Amazon, server log
**Trigger:** A tracked Amazon item shows **Retrieval error** and the operator reads the server log.
**Origin:** Issue #204: every Amazon check failed and the operator could not tell why.
**Covers:** FR-AMZ-BRET-004, FR-AMZ-BRET-005

## Preconditions

- At least one Amazon check has failed.

## Main flow

1. The operator opens the server log.
2. The operator finds the entry of the failed check, with the request URL and time (FR-AMZ-STOP-001).
3. The entry states the cause of the failure in words the operator can act on, one of:
   - the browser could not start, with the browser's own reason;
   - the page could not be opened or did not finish loading;
   - Amazon answered with a non-success status, with the status;
   - Amazon showed a robot check;
   - the page showed no detectable price.
4. The operator acts on the cause (for example installs the missing browser as documented, see UC-AMZ-BRET-003).

## Alternatives and errors

- **Several failures with different causes:** each entry names its own cause; two different causes never produce the same log text.
- **Fifth consecutive failure:** the failure is still logged with its cause, and every tracked Amazon item is stopped as in FR-AMZ-STOP-001–010 (FR-AMZ-BRET-004).
- **Browser not found at start:** the server log says so at start, as today (FR-AMZ-BRET-006).

## Postconditions

- The log contains no tokens or session secrets (FR-AMZ-HUMAN-012).
- Nothing about the failure cause is added to the user interface; item states and messages are unchanged.
