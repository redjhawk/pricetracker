# API contract preservation

Status: ready; unchanged approved contract. Date: 2026-10-03. Agent: `/root/leboncoin_session_technical`.

Canonical contract: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), whose latest listed approved extension is consecutive item price periods on 2026-10-02. The contract's existing LeBoncoin behavior and optional safe `lastAttempt.message` already cover this change.

Both [capture](../../specifications/leboncoin-session-capture/technical.md) and [collection](../../specifications/leboncoin-session-collection/technical.md) designs are ready. Neither adds or changes an HTTP method/path, request or response field, status enum, asynchronous semantics, price representation, schedule or observation-retention rule. Session failures use existing `request_error`; permitted explanatory strings remain safe optional details. Capture/transfer use a local CLI and operator-managed files, not a public session API. Frontend code is unaffected.

`API_SPECIFICATION.md` is therefore intentionally unchanged. The user approved manual desktop verification, protected computer-to-Pi transfer and repeated renewal; this record does not fabricate a separate API approval. Under AGENTS.md and the Go skills, a single-tier change preserving the already approved contract proceeds without another confirmation. Any later proposed contract difference must return to the API stage before implementation.
