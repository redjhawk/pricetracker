# Issue #276 Dependencies: fix the source-map-js advisory reported by npm audit

**Problem:** `npm audit` reports one high-severity advisory in the transitive dependency `source-map-js` 1.0.0–1.2.1 (GHSA-68fv-2mgg-jv7q).

**Suggested work:** Run `npm audit fix` and commit the updated `package-lock.json` on its own, with no other change. `npm audit` must then report no advisory, and the build and tests must still pass.


## Comments by trusted authors

