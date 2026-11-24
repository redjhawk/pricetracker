# Blocking review comments (round 1)

## R1-01 `doc/specifications/test-suite-hygiene/functional.md:3`

Status not updated after implementation. The technical specification (technical.md, 'Development stage' bullet and the 'Files to change' row for issue #295) says the development stage sets FR-TEST-HYG-037–039 and TS-TEST-HYG-044–047 to implemented. In this PR, functional.md line 3 still says 'FR-TEST-HYG-037–039 specified (issue #295)' and technical.md line 3 still says 'TS-TEST-HYG-044–047 ready (issue #295)'. Every earlier issue in these headers (#244, #251, #182, ...) is marked implemented once its change is done. Fix: change functional.md line 3 to 'FR-TEST-HYG-037–039 implemented (issue #295)' and technical.md line 3 to 'TS-TEST-HYG-044–047 implemented (issue #295)'. The test edit itself (tests/platform-tabs.spec.ts:361, expect.poll on imageRequests.length > 0, expect(api.unexpected).toEqual([]) kept) matches TS-TEST-HYG-044/045 and needs no change.

Triage: Confirmed: functional.md line 3 still reads 'FR-TEST-HYG-037–039 specified (issue #295)' and technical.md line 3 'TS-TEST-HYG-044–047 ready (issue #295)', while technical.md line 204 (Development stage) and the 'Files to change' row at line 308 require the development stage to set both to implemented. The technical specification's deliverable is unmet, and merging would leave the spec status wrong; the test change itself (tests/platform-tabs.spec.ts:361) is correct.

