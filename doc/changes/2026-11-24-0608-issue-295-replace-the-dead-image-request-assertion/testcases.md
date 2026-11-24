# Test cases: replace the dead image-request assertion in the TC-010 e2e test (issue #295)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-037 to FR-TEST-HYG-039). No new use cases: see [no-use-cases.md](no-use-cases.md). The application behaviour TC-010 checks is described by UC-AMZ-THUMB-001 and UC-AMZ-THUMB-002, which this change does not alter.

Conventions:

- **TC-010 (e2e):** the Playwright test "TC-010 Amazon list shows the accepted image or the placeholder" in `tests/platform-tabs.spec.ts` (issue #275). It is an existing test, not a case of this catalogue; the IDs below are this change's own.
- **Accepted image URL:** `https://m.media-amazon.com/images/I/price-found.jpg`, the thumbnail TC-010 gives its item "Amazon item with image".
- **Run TC-010:** `npx playwright test tests/platform-tabs.spec.ts -g "TC-010"`.
- The change is test maintenance of one assertion. No production code, API, database or user interface changes, so no Go test and no new Playwright test is written; the cases are repository checks, runs of TC-010 (including temporary fault injections that are reverted afterwards) and the gates.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-037 | TC-001 | — |
| FR-TEST-HYG-038 | TC-002, TC-003 | UC-AMZ-THUMB-001, UC-AMZ-THUMB-002 |
| FR-TEST-HYG-039 | TC-002, TC-004 | UC-AMZ-THUMB-001, UC-AMZ-THUMB-002 |

## Cases

### TC-001 — The always-passing assertion is gone

- Covers: FR-TEST-HYG-037
- Level: repository check (file content)
- Given the change applied on top of `master`
- When `tests/platform-tabs.spec.ts` is searched
- Then it contains no `imageRequests.every(` and no other assertion that compares each recorded image request with the accepted image URL
- Corner cases:
  - the route handler still records only requests for the accepted image URL, so any remaining per-URL comparison of the recorded list would again be always true; none is left;
  - the assertion is not merely commented out (no `imageRequests.every` text at all).

### TC-002 — TC-010 passes on a normal run and asserts the accepted image was requested

- Covers: FR-TEST-HYG-038, FR-TEST-HYG-039; UC-AMZ-THUMB-001, UC-AMZ-THUMB-002
- Level: e2e (Playwright, `tests/platform-tabs.spec.ts`)
- Given the change applied and the e2e environment of the suite
- When TC-010 is run, then run again with `--repeat-each=3`
- Then TC-010 passes every time; its final assertions require the number of accepted image requests to be greater than zero and keep `expect(api.unexpected).toEqual([])`
- Corner cases:
  - all other steps of TC-010 are unchanged and still run: image row shows `img.item-thumbnail` with the accepted URL and no placeholder; placeholder row shows the placeholder and no image; detail page of the item without image has no `img.detail-thumbnail`; back to the list, after `page.reload()`, and at a 320 px wide viewport with no horizontal scroll (`scrollWidth <= 320`);
  - the assertion requires at least one request, not an exact count: after the reload and the new `page.goto("/")` the browser may serve the image from its cache without a new routed request, so a fixed count would make the test flaky;
  - repeated runs give the same result (no dependence on cache state or order).

### TC-003 — TC-010 fails when the accepted image is not requested, and still fails on another image

- Covers: FR-TEST-HYG-038; UC-AMZ-THUMB-001
- Level: e2e (Playwright, temporary fault injection in a local copy of TC-010, reverted after each run and never committed)
- Given the change applied
- When TC-010 is run once with each temporary change:
  1. the route handler fulfils the accepted image URL without recording it (the `imageRequests.push` is removed), so no accepted image request is recorded;
  2. the item "Amazon item with image" gets `thumbnailUrl: null` in the test's mock data, so the image is never requested (the case named in the specification's acceptance criteria);
  3. after the last `expectThumbnails()`, the page loads an image from another URL (for example `https://images-eu.ssl-images-amazon.com/images/I/other.jpg`, or `https://example.com/other.png`);
- Then each run fails: run 1 fails at the new "accepted image requested" assertion (proving that this assertion itself is not dead); run 2 fails (at the thumbnail check or at the new assertion); run 3 fails at `expect(api.unexpected).toEqual([])` with the other image's URL in the reported value; and with the temporary change reverted TC-010 passes again
- Corner cases:
  - run 3 uses an accepted Amazon image host other than the accepted URL, so a request for a different image on an allowed host is still caught as unexpected;
  - no temporary change remains in the diff (checked by TC-004).

### TC-004 — Only TC-010 changes outside `doc/`, and the gates pass

- Covers: FR-TEST-HYG-039
- Level: repository check and suite run (deterministic gates: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`, and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-only master -- . ':!doc'` and `git diff master -- tests/platform-tabs.spec.ts` are listed and the gates run
- Then the only file listed outside `doc/` is `tests/platform-tabs.spec.ts`; every changed hunk of that file lies inside TC-010 and only touches its final image-request assertion; all gates pass, and the Playwright suite runs the same test list as on `master`
- Corner cases:
  - no other test of `tests/platform-tabs.spec.ts`, no shared helper (`mockApi`, fixtures) and no other `.spec.ts` file changes;
  - no production Go code, `_test.go` file, frontend source, `go.mod`, `go.sum`, `package.json` or `API_SPECIFICATION.md` changes, so the operator observes no change in the application;
  - `doc/todo/issue-275-r1-01.md` and the issue #275 change folder are unchanged;
  - the gates give the same result when run twice.
