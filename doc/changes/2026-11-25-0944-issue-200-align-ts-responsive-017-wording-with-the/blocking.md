# Blocking review comments (round 1)

## R1-01 `doc/changes/2026-11-25-0944-issue-200-align-ts-responsive-017-wording-with-the/no-tests.md:11`

The TC-001 check says "`git diff master` on `technical.md` touches only that sentence in the entry". The actual diff on `technical.md` also changes line 4 ("FR-RESPONSIVE-001 to 027" becomes "001 to 030") and adds the issue #200 section (lines 219-236). Run as written, this check fails on the correct change. TC-003 in testcases.md already allows "traceability lines that cite issue #200 or FR-RESPONSIVE-028 to 030". Suggested fix: reword the sentence as "within the TS-RESPONSIVE-017 entry, `git diff master` touches only that sentence", so it matches TC-001's scope ("the rest of the entry is unchanged") and TC-003.

Triage: Confirmed: `git diff master...HEAD -- doc/specifications/responsive-layout/technical.md` also changes line 4 ("001 to 027" to "001 to 030") and adds the issue #200 section, so the TC-001 check in no-tests.md line 11 (and the same wording in testcases.md line 35) fails on the correct change and contradicts TC-003, which allows issue #200 traceability lines. A test check that rejects the correct implementation does not prove its case; scope it to the TS-RESPONSIVE-017 entry in both files.

