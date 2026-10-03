# Functional specification: LeBoncoin session collection

Status: ready  
Owner: functional specification agent  
User decision/reference: The user answered “Yes, manual verification is acceptable” to desktop verification, secure transfer to the Raspberry Pi, and repeating verification when rejected. This subject specifies reuse of that operator-supplied session within the existing collection behavior.

## Purpose and scope

Allow PriceFollower's existing Go collector to reuse a privately installed, manually verified LeBoncoin session for immediate, scheduled, and manual collection attempts. Preserve the current item API, interface, price interpretation, schedule, and observation retention.

Actors are the operator, the running backend, and LeBoncoin. Session-assisted operation requires an operator-configured protected session file produced by the [capture workflow](../leboncoin-session-capture/functional.md). Deployments that do not opt in continue to work as before. The collector continues running on the existing Raspberry Pi deployment without a browser daemon.

Excluded: a new session-management API or UI, automatic CAPTCHA solving, unattended establishment of a verified session, account authentication, altered collection frequency, new retry policies, and guarantees of remote acceptance or session duration.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-LBC-COL-001 | No session file is configured. | Preserve existing sessionless collection behavior for every platform. | Startup requires no browser or session file. LeBoncoin attempts use the existing path and existing error outcomes; Amazon behavior is unchanged. |
| FR-LBC-COL-002 | A protected valid session file is configured. | Use only the applicable LeBoncoin `datadome` session for supported LeBoncoin collection requests. | Immediate, scheduled, and manual attempts can reuse the session. The secret is not sent to Amazon or unrelated hosts, including redirect destinations. No account cookies are imported. |
| FR-LBC-COL-003 | The configured file is missing, malformed, expired, unreadable, or insufficiently protected. | Reject unusable session material safely and make the configuration/renewal problem understandable to the operator without exposing its contents. Keep the backend available. | Unsafe cookie data is never sent. A collection that cannot proceed records the existing `request_error` outcome; previous observations and unrelated collection continue to function. Diagnostics identify the condition without printing secrets. |
| FR-LBC-COL-004 | A legitimate response updates or expires the LeBoncoin session cookie. | Retain the applicable update safely for later requests and across backend restarts; preserve private file protection. | A later attempt and a restart use the updated session state. A removed/expired cookie is not silently resurrected from an older saved value. Unrelated response cookies are not persisted. |
| FR-LBC-COL-005 | The operator installs a renewed valid session file while the backend is running. | Load the replacement for subsequent collection attempts without requiring a server restart. | After safe replacement, a new attempt uses the replacement session. An earlier in-flight attempt does not overwrite the operator's newer session with stale state. |
| FR-LBC-COL-006 | Session state cannot be saved. | Report the persistence problem safely and leave the installed session file intact; do not invent or discard a successfully parsed price because of a session-file write problem. | The operator receives a sanitized diagnostic. A complete previous export remains readable, and a successful listing response can still create its normal observation. No truncated session file is installed. |
| FR-LBC-COL-007 | LeBoncoin accepts a session-assisted request. | Apply the existing listing-identity, availability, and price-validation rules before recording success. | Only a valid response for the requested listing creates its normal observation. A cookie is not proof of a correct listing or price, and collection retains existing euro-cent semantics. |
| FR-LBC-COL-008 | LeBoncoin returns a challenge, 403, or another existing request-error condition. | Record `request_error`, preserve the latest successful price and all prior observations, and continue existing retries. | The listing is not marked free or unavailable merely because verification is requested. Existing API and UI show the established collection error/stale-price behavior. Operator guidance points to manual renewal where appropriate. |
| FR-LBC-COL-009 | Collection, restart, or operator troubleshooting occurs. | Keep session secrets out of application responses, logs, repository content, and diagnostic evidence. | Existing API responses expose no session-file contents or cookie values. Diagnostics can explain load, rejection, renewal, or persistence failure without disclosing the secret. |
| FR-LBC-COL-010 | A desktop session is imported on the Raspberry Pi. | Treat a successful server response as the evidence of usable collection; do not treat import alone as proof of acceptance. | The operator can check an attempt using existing controls. A cross-network rejection remains a normal collection failure with documented manual renewal; neither capture nor import claims guaranteed remote success. |

## States and corner cases

The existing application states remain pending collection, successful observation, unavailable listing, price not found, and request error according to the approved contract. Session capture/import does not create new client-visible fields or status values. Last successful prices remain available when session-assisted attempts fail.

Session-file absence when no file is configured is ordinary operation; a configured but unusable file is an operator-visible configuration/renewal problem. A renewal takes effect on a subsequent attempt, not retroactively on an already completed request. Concurrent attempts and file replacement must not corrupt the session file or restore superseded session data. Session updates do not change SQLite price history or scheduling rules.

## Open questions

None affecting this handoff. File format, configuration names, safe persistence mechanism, and exact diagnostics belong to technical specification. Existing product questions in the general functional document, such as staleness thresholds, are unchanged and are not prerequisites for this subject.

## Traceability

- [Existing functional specifications](../../FUNCTIONAL_SPECIFICATIONS.md), section 6 and FR-04/FR-07/FR-14/FR-18.
- [Canonical API contract](../../../API_SPECIFICATION.md): preserve existing collection and error semantics.
- [Investigation](../../changes/leboncoin-403-investigation/report.md): verified local cookie reuse succeeded, remote portability remains unproven.
- [Capture subject](../leboncoin-session-capture/functional.md).
- Technical handoff target: `doc/specifications/leboncoin-session-collection/technical.md`.
