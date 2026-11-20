# UC-AMZ-HCHK-004 — Restart stopped Amazon items once after the upgrade

**Primary actor:** PriceFollower application, on behalf of the user who tracks Amazon items
**Supporting systems:** SQLite store, PriceFollower scheduler
**Trigger:** The application starts for the first time after this change is installed.
**Origin:** Issue #168: all Amazon items were stopped on 20 Nov 2026, 07:26 UTC; the user chose an automatic one-time restart (answer 4).
**Covers:** FR-AMZ-HCHK-010

## Preconditions

- One or more tracked Amazon items are stopped (FR-AMZ-STOP-003), and the Amazon tab shows the stop warning.
- The one-time restart has never run.

## Main flow

1. On first start after installation, the application restarts every stopped Amazon item: the stopped state is cleared and the consecutive failure count is reset to 0.
2. Price histories and past check outcomes are kept.
3. The application records that the one-time restart has run.
4. The user opens the Amazon tab: the stop warning is gone and no item shows as stopped.
5. Each item gets its next scheduled check at its random time in the next window (UC-AMZ-HCHK-001).

## Alternatives and errors

- **No stopped Amazon item:** nothing is changed; the one-time restart is still recorded as done.
- **Later application restart:** the restart does not run again; an item stopped after installation stays stopped (FR-AMZ-STOP-010) and is restarted only as in [Amazon request stop](../../specifications/amazon-request-stop/functional.md) (Refresh price).
- **Protected mode:** the stopped Amazon items of all users are restarted.

## Postconditions

- No Amazon item stopped before the change remains stopped.
- Later stops follow [Amazon request stop](../../specifications/amazon-request-stop/functional.md) unchanged.
