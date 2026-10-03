# Functional specification: LeBoncoin session capture

Status: ready  
Owner: functional specification agent  
User decision/reference: The user answered “Yes, manual verification is acceptable” to completing the verification in a browser on their computer, securely transferring the verified session to the PriceFollower Raspberry Pi, and repeating this when rejected. This approval covers that workflow; it does not approve an HTTP contract change.

## Purpose and scope

Let the operator establish a verified LeBoncoin session on their desktop computer and transfer the session material needed by the existing Go collector to the Raspberry Pi. The operator performs any interactive verification. A command-line helper and documented secure transfer/install steps implement the coordinator's selected workflow; there is no new application screen or HTTP endpoint.

The actors are the operator, the desktop helper, LeBoncoin, and the operator's existing Raspberry Pi deployment. Prerequisites are a supported visible desktop browser, access to a supported LeBoncoin listing, and the operator's existing permission to install the exported file on their server. Browser dependencies and commands belong in the technical specification and operator documentation.

Excluded: account login, personal browser profile access, account-cookie export, automated CAPTCHA solving, paid services, proxies, a mandatory browser process on the Raspberry Pi, or guarantees that a desktop session will be accepted from a different device/network.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-LBC-CAP-001 | The operator starts capture with a listing URL and output destination. | Accept an eligible HTTPS LeBoncoin France listing using the application's supported listing rules; explain invalid input before attempting capture. | A supported listing can be used without editing source code. Invalid schemes, unrelated hosts, and non-listing URLs fail clearly. The investigation's listing ID is not a hardcoded restriction. |
| FR-LBC-CAP-002 | Valid capture starts. | Open a visible, isolated browser session that does not read or modify the operator's personal browser profile. Explain that the operator completes any LeBoncoin verification themselves. | The operator sees the requested listing or its verification page in a dedicated browser session. Personal login/session state is not required. |
| FR-LBC-CAP-003 | LeBoncoin requests interaction. | Wait for the operator to complete verification or cancel, with a bounded waiting period and an understandable progress message. | No automation solves or submits the interactive challenge. Cancellation or timeout terminates the helper with a clear unsuccessful outcome. |
| FR-LBC-CAP-004 | The browser appears to have completed verification. | Verify that the requested listing has actually loaded and its listing identity matches the input before exporting. Require the applicable `datadome` cookie. | A challenge page, wrong listing, navigation error, or missing cookie cannot produce a successful export. A matching loaded listing with the cookie can produce an export. The presence of an unverified cookie alone is insufficient. |
| FR-LBC-CAP-005 | Capture succeeds. | Export only the LeBoncoin `datadome` session cookie and the metadata needed to reuse it safely. | The export contains no account/login cookies, personal profile, page HTML, unrelated cookies, or collected price history. Cookie values do not appear in terminal output, logs, source control, or workflow evidence. |
| FR-LBC-CAP-006 | An export is written or renewed. | Protect session material against access by other local users; reject unsafe destinations and preserve an existing export if capture or writing fails. Permit deliberate renewal of the operator's session file. | The resulting file is private to its owner. Invalid/unwritable destinations and unsafe file targets fail clearly. Cancellation, a rejected challenge, and incomplete writes do not replace a usable previous export. |
| FR-LBC-CAP-007 | Capture exits successfully or unsuccessfully. | Release the helper's browser resources and dispose of its temporary session data when no longer needed. | Normal success, cancellation, and handled failure leave no required background browser service or reusable temporary personal-data export beyond the intended private session file. |
| FR-LBC-CAP-008 | The operator installs or renews the session on the Raspberry Pi. | Provide explicit instructions to securely transfer and privately install the export for the running service, configure its use, and check a normal collection through existing application controls. | The documented computer-to-Pi sequence identifies where the protected session file belongs and how to replace it safely. It does not ask the operator to paste the secret into a URL, application form, command argument, or public log. No new HTTP route is required. |
| FR-LBC-CAP-009 | The server rejects a transferred session or a previously working session stops working. | Explain that the operator repeats manual verification and installation and that cross-device/network acceptance and session lifetime are not guaranteed. | Documentation distinguishes desktop capture success from successful server collection. It gives a repeatable renewal procedure and does not promise unattended verification or permanent access. |

## States and corner cases

Capture states are validating input, opening the browser, waiting for human verification/listing access, exporting, and completed or failed. Text feedback must distinguish these states without revealing the cookie; no new Carbon interface is introduced. The helper's own commands and cancellation must be keyboard usable. Accessibility of LeBoncoin's third-party challenge is outside this application's control.

A listing that is removed or otherwise cannot be verified as the requested loaded listing does not establish capture success. The operator can choose another supported listing. A successful capture does not itself add an item, collect a backend price, or prove that the Raspberry Pi will be accepted. The existing tracked-item workflows remain the means to observe server-side results.

## Open questions

None affecting this handoff. File format, browser integration, timeout value, exact commands, and transfer mechanics are technical choices within this scope. No unrelated refactoring is requested.

## Traceability

- [Existing functional specifications](../../FUNCTIONAL_SPECIFICATIONS.md), sections 4–6 and FR-07/FR-10.
- [Investigation and demonstrated limitations](../../changes/leboncoin-403-investigation/report.md).
- [Session collection subject](../leboncoin-session-collection/functional.md).
- Technical handoff target: `doc/specifications/leboncoin-session-capture/technical.md`.
