# Red phase skipped for issue #295

This change is the test code itself. TC-010 in `tests/platform-tabs.spec.ts` tests behaviour that issue #275 already shipped, and the technical specification (`doc/specifications/test-suite-hygiene/technical.md`, "TC-010 proves the accepted image is requested") says no production code changes. TC-010 passes on the `master` code before and after the edit, so the red gate would always fail. This file skips it. The development stage writes no code.

The fault injections below show that the new assertion can fail.

## Test edited

In TC-010 `Amazon list shows the accepted image or the placeholder`, the always-passing line

```ts
expect(imageRequests.every((url) => url === imageUrl)).toBe(true);
```

is replaced, in place, by

```ts
await expect.poll(() => imageRequests.length, { message: "the accepted Amazon image was requested" }).toBeGreaterThan(0);
```

`expect(api.unexpected).toEqual([])` stays right after it. No other line of the file changes. No test is added or removed.

| Test case | Proved by |
| --- | --- |
| TC-001 | `git grep -n 'imageRequests' -- tests/platform-tabs.spec.ts` lists only the declaration (line 323), the `push` in the route handler (line 328) and the poll on `imageRequests.length` (line 361). No `imageRequests.every` remains, commented out or otherwise. |
| TC-002 | `npx playwright test tests/platform-tabs.spec.ts -g "TC-010"` passes, and so does `--repeat-each=3` (3 passed). |
| TC-003 | Fault injections 1–3 below. |
| TC-004 | `git diff --name-only master -- . ':!doc'` lists only `tests/platform-tabs.spec.ts`. `git diff master -- tests/platform-tabs.spec.ts` has one hunk inside TC-010 that replaces the one line. The full `npx playwright test` run passes (156 passed, the same test list as `master`). |

## Fault injections (TC-003)

Each was a temporary edit of TC-010, then `npx playwright test tests/platform-tabs.spec.ts -g "TC-010"`, then a revert. None is in the diff.

1. **No accepted request recorded.** `imageRequests.push(request.url());` removed from the route handler.
   - **FAIL** at line 360, the new poll: `Error: the accepted Amazon image was requested`, `expect(received).toBeGreaterThan(expected)`, `Expected: > 0`, `Received: 0`, `Timeout 5000ms exceeded while waiting on the predicate`.
2. **No thumbnail.** `withImage` given `thumbnailUrl: null`.
   - **FAIL** at line 339, the `src` check of `expectThumbnails`: `expect(locator).toHaveAttribute(expected) failed`, `Expected: "https://m.media-amazon.com/images/I/price-found.jpg"`, `element(s) not found`.
3. **Another Amazon image.** After the last `expectThumbnails()`:

   ```ts
   await page.evaluate((url) => new Promise<void>((done) => { const image = new Image(); image.onload = image.onerror = () => done(); image.src = url; }), "https://images-eu.ssl-images-amazon.com/images/I/other.jpg");
   ```

   - **FAIL** at `expect(api.unexpected).toEqual([])`: received `["image https://images-eu.ssl-images-amazon.com/images/I/other.jpg"]`. The poll before it passed.
4. **Revert.** After the last revert, `npx playwright test tests/platform-tabs.spec.ts -g "TC-010" --repeat-each=3` gives 3 passed.
