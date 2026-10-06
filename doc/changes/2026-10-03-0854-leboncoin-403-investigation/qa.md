# Independent QA: LeBoncoin access investigation

Status: executed; diagnostic verification passed. Date: 2026-10-03.
Tester: `/root/leboncoin_investigation_qa`, independent of the experiment coordinator, probe developer, reviewer and adjudicator.

Scope: verify the reported acquisition method for the user-supplied public listing and the supporting documentation. This investigation implements no application behavior. Product interface journeys, randomized UI actions and new feature tests are not applicable; no application base URL was started or visited by this tester. External HTTP verification below is not represented as application interface testing.

Repository base: `fe76586952fac12a5aa0c884172a5b8e16a38364`. Verified [report](report.md) SHA-256: `e5cce9730b934848819b499c4afc85b72e608b8583547027d842a5f9e1a213ed`, matching the final revision in [independent review](review.md). Read AGENTS.md, the QA role, all four required skills, workflow, the investigation index, report, review and [decisions](decisions.md), and relevant [approved API](../../../API_SPECIFICATION.md) clauses. REV-001 is corrected and independently verified in both review records; no unresolved critical finding remains.

## Environment and exact sequence

Local Linux/amd64 host, Go 1.25.0. The task-owned probe directly invokes the repository's `leboncoin.NewCollector(...).Collect(...)`; it does not start the service or access SQLite. Source copies `.tmp-leboncoin-probe/main.go` and `/tmp/pricefollower-leboncoin-diagnostic/go-probe.go` both hash to `34077baa6349cc90a0449ed5578fa488860594426f65c51cdcc5e5a33f17af22`. The executable's build metadata identifies the same module path, base revision and Go version.

Actual external URL for both executed cases: <https://www.leboncoin.fr/ad/voitures/3245888872>. This is the source listing URL, not a PriceFollower application route. Run order was fixed, with no random seed: baseline once, then cookie154 once. Scoped network execution was approved. No new browser interaction or human verification was requested.

```bash
/tmp/pricefollower-leboncoin-diagnostic/go-probe baseline
/tmp/pricefollower-leboncoin-diagnostic/go-probe cookie154
```

A Python subprocess wrapper captured each command's stdout, stderr and exit code, parsed the JSON and transport status, and asserted the fields below. It stopped on any failed assertion. Each subprocess had a 40-second outer timeout. Sanitized captured stdout objects and exact stderr strings are retained in `/tmp/pricefollower-leboncoin-diagnostic/qa-go-results.json`; this is independent execution evidence, not a transcription of the coordinator's results.

## Executed cases

| Case | Requirement / corner case | Preconditions and actions | Expected | Actual result and evidence |
| --- | --- | --- | --- | --- |
| QA-001 | Reproduce LBC-12; unverified collector request | Exact URL above; current Chrome154 headers; baseline command with no cookie | One request; HTTP 403; `request_error` | At `2026-10-03T08:51:08.272268184Z`, one HTTP/2.0 request returned 403, `request_error`, amount 0, null title. Exit 0 and explicit assertions passed. This is an expected failed collection, not a successful acquisition. |
| QA-002 | Reproduce LBC-14; verified-session reuse | Same exact URL and headers; cookie154 command reads only the existing validated `datadome` cookie | One request; HTTP 200; `success`; 2690000 cents; exact expected title; matching active listing | At `2026-10-03T08:51:08.999815241Z`, one HTTP/2.0 request returned 200, `success`, 2690000 cents and `Renault Symbioz 1.8 E-Tech full hybrid 160ch Evolution - 25`. Exit 0 and explicit status/result/amount/title/count assertions passed. Listing identity and active status follow the independently inspected collector success path described below. |
| QA-003 | Diagnostic source scope and listing identity | Inspect both matching probe sources and the actual collector | Real collector; fixed requested identity; bounded exact-host requests; no persistence or secret output | Passed. `ParseURL` derives ID `3245888872` from the fixed target and populates `model.Listing.ListingID`. The collector rejects a different `ad.list_id` or non-active status before reporting success. The probe JSON has no ID field; no direct output assertion of an absent field is claimed. Its transport rejects other schemes/hosts/userinfo, removes existing Cookie headers, and injects only `datadome`. Collector logs are suppressed; the structured result contains no cookie value. |
| QA-004 | ARMv6 build evidence | Inspect existing binary using `file` and `go version -m`; inspect local `go version` | Static ARM binary with recorded Go1.25 toolchain and ARMv6 settings | Passed as artifact inspection: ELF 32-bit ARM EABI5, statically linked; metadata reports Go1.25.0, `GOOS=linux`, `GOARCH=arm`, `GOARM=6`, `CGO_ENABLED=0` and the base revision above. Local `go version` reports `go1.25.0 linux/amd64`. QA did not rebuild or execute this ARM binary. |
| QA-005 | Documentation and change boundaries | Compare report hash to final reviewer hash; inspect adjudication; inspect Git diff/status and local Markdown links | Reviewed corrected report; resolved finding; no tracked production/API/dependency changes; local links resolve | Passed. `git diff --exit-code` and `git diff --cached --exit-code` were empty/successful. Only investigation documentation and the temporary module-local probe were untracked. Local relative links in the investigation Markdown files resolve. |
| QA-006 | Credential-file handling | Inspect file modes without printing cookie contents | Private directory and cookie export | Passed: diagnostic directory mode 0700, `validated-cookies.json` mode 0600. No credentials, challenge identifiers or network addresses were copied into this report. |

## Coverage, limits and handoff

The independent live pair supports short-term reuse of this browser-validated session by the existing Go collector on this host/network. Process exit code alone was not treated as acquisition success. No additional network requests were needed after the two passing cases, and no application/database mutation occurred.

The earlier curl, browser, Lynx and TLS experiments remain coordinator execution evidence inspected by the independent reviewer; QA did not repeat them or witness the original human verification. No remote ARMv6 execution, cross-device/IP cookie transfer, cookie lifetime, scheduled durability, unattended renewal or other listing/category coverage is established. The saved offline parser case was not rerun by QA. These are disclosed investigation limits, not verified production features.

No new QA finding. The coordinator may remove the module-local temporary probe and proceed with the documentation-only [commit stage](commit-step.md). Temporary evidence and private session artifacts remain outside the documentation commit and may expire. QA does not claim that the commit has already succeeded.
