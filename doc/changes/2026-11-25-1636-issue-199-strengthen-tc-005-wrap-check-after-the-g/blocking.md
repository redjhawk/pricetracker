# Blocking review comments (round 1)

## R1-01 `doc/changes/2026-11-25-1636-issue-199-strengthen-tc-005-wrap-check-after-the-g/no-tests.md:59`

The gate evidence is missing. The line says the results of `npx playwright test`, `go vet ./...` and `go test ./...` are 'see below', but the file ends there. The TC-007 row (line 38) points to this 'Gates' section as its proof. FR-TEST-HYG-098, TS-TEST-HYG-127 and TC-007 require the e2e suite, `go vet ./...`, `go test ./...` and `node --test scripts/*.test.mjs` to pass, and run 5 of the design requires two runs. Only one `node --test` run is recorded, and `npx tsc` was skipped. Fix: run the missing gates, twice as the design asks, and record each command with its result, or remove the dangling 'see below' and say which gates were not run.

Triage: Confirmed: no-tests.md line 59 says the results of `npx playwright test`, `go vet ./...` and `go test ./...` are 'see below', but the file ends at line 61. The TC-007 row (line 38) cites this 'Gates' section as its proof, and TC-007 and TS-TEST-HYG-127 require these gates to pass twice. Only one `node --test` run is recorded, so the proof of TC-007 is missing.

## R1-02 `doc/changes/2026-11-25-1636-issue-199-strengthen-tc-005-wrap-check-after-the-g/no-tests.md:47`

TC-002 is not proved as specified. TC-002 (testcases.md line 48), TS-TEST-HYG-128 and design run 2 (technical.md line 891) all say that with the 40rem injection 'the `master` test passes'. The record shows otherwise: the `master` test fails at 1055 px on 'Prices and Status on the same line'. The record notes the mismatch but leaves the test case and specification unchanged, so the delivered proof contradicts the test case it claims to prove. Fix, either: (a) make the run 2 injection only affect 775 px, for example by putting the 40rem rule in an `@media (max-width: 800px)` block, so that the `master` test passes and TC-002 holds as written; or (b) reword the last Then clause of TC-002, TS-TEST-HYG-128 and design run 2 to 'with the same injection the `master` wrap check passes every 775 px card'. Run 3 (100%) already gives a full `master` pass, so the gap is still shown.

Triage: Confirmed: TC-002 (testcases.md line 48), TS-TEST-HYG-128 and design run 2 (technical.md lines 891 and 1151) say that with the 40rem injection the `master` test passes. But no-tests.md run 2 (line 47) records that the `master` test fails at 1055 px on 'Prices and Status on the same line'. The delivered proof contradicts the test case it claims to prove. Either the injection or the TC-002/spec wording must change.

