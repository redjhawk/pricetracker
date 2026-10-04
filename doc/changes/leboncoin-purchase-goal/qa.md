# QA execution: leboncoin-purchase-goal

Status: blocked
Tested code/specification revisions: 8f420ca (branch `ai-dev/issue-10-20261004-0954`); [functional](../../specifications/leboncoin-purchase-goal/functional.md) FR-PURCHASE-GOAL-001–009
Environment: `npm run dev`. Vite reported `http://localhost:5173/`, and the API logged `PriceFollower listening on http://127.0.0.1:3001`. Development seed data was written to `.data/pricefollower.sqlite` ("Seeded 6 sample items"). Chromium via Playwright 1.63.0 was planned.
Executed at: 2026-10-04 ~10:16 UTC (server start only)
Tester: QA tester agent (stage 7)

## Exploratory session

Not executed. The dev server started successfully. The permission system then refused every command needed to drive it: `node <script>`, `npx playwright`, and `curl` against the API each returned "This command requires approval". No approval was available in this non-interactive run. Working around the refusal (for example by routing an arbitrary script through an approved npm script) would bypass the permission system, so it was not attempted. No browser page was loaded and no API request was sent by the tester.

## Executed cases

| Case | Requirement / corner case | Actual interface URL | Actual source listing URL, if relevant | Preconditions and exact actions | Expected | Observed | Result / evidence / finding |
| --- | --- | --- | --- | --- | --- | --- | --- |
| — | — | — | — | — | — | — | No cases executed (blocked). |

## Proposed cases (not executed)

The cases below were scripted but never run. They use the seeded LeBoncoin item (`sample-lbc-*`) and a seeded Amazon item.

- FR-001: open the add-item modal and check that "Purchase goal (optional)" is a textarea. Mock `POST /api/v1/items` with `page.route` to capture the request body containing `purchaseGoal`. A real LeBoncoin add depends on scraping, which may be unavailable.
- FR-002/003: on `/items/<lbc-id>`, check the field starts empty, then set a goal, save, and reload. Also save an unchanged goal and confirm Save is disabled.
- FR-003/006: save a whitespace-only goal and confirm it is stored as empty. Clear a goal, save, and reload.
- FR-007: save a multi-paragraph goal of about 10k characters, reload, and confirm it is shown in full. Check that a body over 1 MiB returns `413 REQUEST_TOO_LARGE`.
- FR-008: `/items/<amazon-id>` has no goal field.
- FR-009: reach the field with Tab, type, Tab to "Save goal", press Enter, and confirm the success announcement. Force a 500 with `page.route` and confirm the error notification appears and the entered text is kept.
- API: `PUT /api/v1/items/nope/purchase-goal` returns 404 `ITEM_NOT_FOUND`. The same PUT on an Amazon item returns 422 `PURCHASE_GOAL_UNSUPPORTED`. Malformed JSON, a missing field, a non-string value, or two JSON values return 400 `INVALID_JSON`.
- Narrow screen: at a 320 px viewport, check for horizontal overflow and that the field and Save button stay visible.
- FR-004/005: needs a Claude token. If none is available, check that `reviewStarted` is `false`.

## Coverage and limitations

Coverage is zero. Interface and API behavior is unverified. Automated Playwright results from development ([implementation.md](implementation.md)) are not independent QA.

## Handoff and cleanup

No findings were raised. The dev server was stopped and the temporary script was deleted. The development seed database in `.data/` was created by the dev server; the tester mutated no data. To unblock, re-run stage 7 with permission to run `node`/`npx playwright` and `curl` against localhost.
