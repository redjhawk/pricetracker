# Blocking review comments (round 1)

## R1-01 `doc/specifications/test-suite-hygiene/functional.md:694`

The issue #228 acceptance criterion for FR-TEST-HYG-138 says the index line and the two status lines contain neither `specified` nor `ready` with "`grep -c` on each gives 0". Without `-w` this check fails on the delivered state: the index row still holds the word `already` twice, in the unchanged #238 and #359 notes. Running `grep '^| Test suite hygiene |' doc/FUNCTIONAL_SPECIFICATIONS.md | grep -c ready` gives 1, not 0. testcases.md (TC-004), no-tests.md and the technical mapping all match whole words (`grep -cwE 'specified|ready'`), so the functional acceptance criterion contradicts the test case that proves it. Fix: write `grep -cwE 'specified|ready'` on each gives 0 (whole words), and say "whole word" in the FR-TEST-HYG-138 text at line 431 too.

Triage: Verified: `grep '^| Test suite hygiene |' doc/FUNCTIONAL_SPECIFICATIONS.md | grep -c ready` gives 1 (the `already` notes of #238 and #359), while `grep -cwE 'specified|ready'` gives 0. The acceptance criterion at functional.md:694 (plain `grep -c`) and the FR-TEST-HYG-138 text at line 431 therefore state a check that fails on the delivered state and contradicts TC-004 and TS-TEST-HYG-183, which use whole-word matching.

