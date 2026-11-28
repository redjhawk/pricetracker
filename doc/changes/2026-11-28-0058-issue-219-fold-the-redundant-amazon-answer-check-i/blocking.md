# Blocking review comments (round 1)

## R1-01 `doc/specifications/test-suite-hygiene/technical.md:3`

The Status line still says `TS-TEST-HYG-145–149 specified (issue #219)`. The issue #219 design (line 1048, 'Development stage') and the file table (line 1332) say the development stage sets TS-TEST-HYG-145–149 to implemented. The functional spec and `doc/FUNCTIONAL_SPECIFICATIONS.md` already say FR-TEST-HYG-111–114 are implemented, so the documents now disagree. Fix: change the entry to `TS-TEST-HYG-145–149 implemented (issue #219)`.

Triage: Confirmed: line 3 of technical.md still reads `TS-TEST-HYG-145–149 specified (issue #219)`, while the design (line 1048) and the file table (line 1332) require the development stage to set them to implemented, and functional.md and FUNCTIONAL_SPECIFICATIONS.md already say FR-TEST-HYG-111–114 are implemented. A required delivery step is missing and the documents disagree.

## R1-02 `doc/specifications/test-suite-hygiene/functional.md:579`

This acceptance criterion says `grep -rn '"not a failure: ' internal/amazon/` prints nothing, but `no-tests.md` shows it prints six lines from files this change must not touch (`browser_interrupted_test.go:59`, `browser_test.go:364/383`, `browser_retrieval_test.go:417/568`, `browser_profile_test.go:136`). FR-TEST-HYG-114 limits the diff to `browser_navigation_test.go`, so the criterion cannot be met and contradicts the requirement. The tests stage worked around it by 'reading the grep as limited to this file' instead of the specification saying so. TC-002 in `testcases.md` (line 42) and run 3 of the technical design (line 1042) have the same grep. Fix: limit the grep to `internal/amazon/browser_navigation_test.go` in all three places so the documented proof matches what was run.

Triage: Confirmed: `grep -rn '"not a failure: ' internal/amazon/` prints six lines (browser_test.go:364/383, browser_profile_test.go:136, browser_interrupted_test.go:59, browser_retrieval_test.go:417/568), so the acceptance criterion at functional.md:579, TC-002 (testcases.md:42) and run 3 (technical.md:1042) cannot hold while FR-TEST-HYG-114 forbids touching those files. The specification marked implemented states a proof that fails; the grep must be limited to `internal/amazon/browser_navigation_test.go` in all three places.

