# Test problem: TC-017 e2e tests do not await `page.evaluate`

Locked file: `tests/leboncoin-session-settings.spec.ts`

Failing tests (always, with the implementation in place; the other 147 Playwright tests and all Go tests pass):

- `TC-017 (#265) the browser holds no full secret after the modal closes` (line 726)
- `TC-017 (#265) narrow viewport and page reload give the same result` (line 742)

Error, both tests, at the last `expectNoTypedSecret` of `secretLifecycle` (line 723):

```
Error: page.evaluate: Test ended.
    at browserState (tests/leboncoin-session-settings.spec.ts:682:15)
```

## Cause

`browserState` (lines 681–688) is:

```ts
return page.evaluate(async () => { ... }) + JSON.stringify(await page.context().cookies());
```

`page.evaluate(...)` is not awaited, so it is a Promise concatenated with a string. `browserState` therefore returns `"[object Promise]" + cookies` without waiting for the page state, and the evaluation keeps running in the background:

1. The check is vacuous: `expect(state).not.toContain(secret)` never sees storage, IndexedDB, form values, the DOM or the URL.
2. After the last call (line 723) the test only runs synchronous assertions and ends. The fixture closes the browser context ("Test ended.") while the floating `page.evaluate` is still waiting on `await indexedDB.databases()`. Its rejection fails the test. This race does not depend on the application code. Earlier calls only pass because the next awaited Playwright action gives the evaluation time to finish.

## Fix needed in the test

Await the evaluation:

```ts
async function browserState(page: Page) {
  const pageState = await page.evaluate(async () => { /* unchanged */ });
  return pageState + JSON.stringify(await page.context().cookies());
}
```

With that fix the assertions check what TC-017 intends. The implementation keeps no typed secret: inputs reset to `""` on close, and nothing is written to `localStorage`, `sessionStorage`, IndexedDB, cookies or the URL. I could not run the fixed version because developers may not edit locked test files.
