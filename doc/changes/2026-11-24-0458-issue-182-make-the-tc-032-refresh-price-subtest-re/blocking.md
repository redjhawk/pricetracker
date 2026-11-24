# Blocking review comments (round 1)

## R1-01 `doc/specifications/test-suite-hygiene/functional.md:3`

The development stage did not update the statuses. The technical specification (Scheduled checks around a Refresh price, "Development stage", and the Files to change table) says it sets FR-TEST-HYG-033–036 and TS-TEST-HYG-038–043 to implemented. The development commit 4bd4a63 only touches state.json, so the PR merges with `FR-TEST-HYG-033–036 specified (issue #182)` here, `TS-TEST-HYG-038–043 ready (issue #182)` in doc/specifications/test-suite-hygiene/technical.md line 3, and `FR-TEST-HYG-033–036 specified (issue #182, …)` in the Test suite hygiene row of doc/FUNCTIONAL_SPECIFICATIONS.md (line 358). Every earlier issue in this spec (#181, #185, #238, …) is recorded as implemented after its merge. Fix: change the three status texts to implemented.

Triage: Confirmed: technical.md line 169 and the Files to change row (line 265) require the development stage to set FR-TEST-HYG-033–036 and TS-TEST-HYG-038–043 to implemented, but functional.md line 3 still says 'specified (issue #182)' and technical.md line 3 'ready (issue #182)', unlike every earlier issue in this spec. The doc/FUNCTIONAL_SPECIFICATIONS.md row does not need changing: every issue in that row, including the implemented ones, is recorded as 'specified'.

