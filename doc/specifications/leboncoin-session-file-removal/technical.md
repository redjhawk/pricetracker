# Technical specification: file-based LeBoncoin session removal

Status: ready (removal is part of the cross-tier change; API approved by the user 2026-10-03 (proposal revision 1 unchanged; see the change index and api-step.md); implementation authorized)\
Functional specification: [functional.md](functional.md), FR-LBC-RM-001 – 007 (ready)\
Companions: [collection rev. 2](../leboncoin-session-collection/technical.md) (TS-LBC-COL-006), [capture rev. 2](../leboncoin-session-capture/technical.md) (TS-LBC-CAP-007, 011), [settings](../leboncoin-session-settings/technical.md)

## Requirement mapping

`RM-*` abbreviates `FR-LBC-RM-*`.

| Technical ID | Functional IDs | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- | --- |
| TS-LBC-RM-001 | RM-001, RM-002 | Delete `Config.LeboncoinSessionFile`, its env read and its test; delete file/sidecar code; no code reads or deletes old files | `config/config.go`, `config/config_test.go` (delete), `internal/leboncoin/session.go`, `internal/leboncoin/collector.go`, `internal/service/service.go`, `internal/leboncoin/session_test.go`, `internal/service/session_test.go` | Build; grep; startup with the variable set to a valid, a missing and no file |
| TS-LBC-RM-002 | RM-003 | Remove `--output`, `validateDestination`, `writeSession` from the helper | `scripts/capture-leboncoin-session.mjs`, its test | TS-LBC-CAP-012 tests |
| TS-LBC-RM-003 | RM-004 | Rewrite operator guide; update README link text; no live reference left | `doc/leboncoin-session.md`, `README.md` | grep below |
| TS-LBC-RM-004 | RM-005 | New `doc/deprecated/README.md` and `doc/deprecated/leboncoin-session-file.md`; link from `doc/README.md` | new files, `doc/README.md` | Link check; content checklist |
| TS-LBC-RM-005 | RM-006 | Do not edit `doc/changes/2026-10-03-1213-leboncoin-session/` or `doc/changes/2026-10-03-0854-leboncoin-403-investigation/`; superseded rows/sections stay marked | — | `git diff --stat` on those directories is empty |
| TS-LBC-RM-006 | RM-007 | Item API, outcomes, schedule, Amazon unchanged | — | Existing Go tests and `tests/platform-tabs.spec.ts` pass |

## Frontend

Not affected: the old mechanism had no UI. The replacement UI is in the [settings](../leboncoin-session-settings/technical.md) and [header menu](../app-header-menu/technical.md) subjects.

## Backend

TS-LBC-RM-001, exact removal list:

- `config/config.go`: field `LeboncoinSessionFile` and `LEBONCOIN_SESSION_FILE` read. Nothing else in `Load` changes. The variable is simply not read, so a leftover systemd drop-in has no effect (RM-001). No startup warning (functional decision: not requested).
- `config/config_test.go`: deleted (its only test covers the removed variable).
- `internal/leboncoin/session.go`: delete `sessionLimit`, `errSession*`, `sessionState`, `sessionStore` and its `load`/`save`/`finish`, `exactObject`, `utcTimestamp`, `decodeCookie`, `owned`, `privateSessionDirectory`, `readPrivate`, the fingerprint and sidecar logic. Keep and adapt `validCookie`, `allowedSessionURL`, `cookieApplies`, `expired`, the `sessionAttempt` jar (`Cookies`, `SetCookies`, `sameIdentity`) per TS-LBC-COL-009, and add the parser/types of TS-LBC-SET-002 and TS-LBC-COL-009.
- `internal/leboncoin/collector.go`: delete `NewCollectorWithSession`, the `session` field, the gate and `sessionCollectionError` (“private session files” text).
- `internal/service/service.go`: replace `NewCollectorWithSession(cfg.UserAgent, cfg.LeboncoinSessionFile)` per TS-LBC-COL-006.
- Tests `internal/leboncoin/session_test.go` and `internal/service/session_test.go`: file/sidecar/permission cases removed; replaced by the store-backed cases of the collection design.
- No code opens, imports or deletes `session.json`, `*.state.json` or any staging directory on existing servers (RM-002). The database starts with `leboncoin_session.value = NULL`, so the modal shows no session after upgrade.
- `scripts/install-pricefollower.sh`, `scripts/deploy-armv6.sh`, `DEPLOYMENT.md`: verified to contain no session references; unchanged.

## API

Not affected: the file mechanism had no API. Existing item contract unchanged (RM-007). The new settings endpoints are proposed in the settings subject.

## Documentation

TS-LBC-RM-003: `doc/leboncoin-session.md` rewritten per TS-LBC-CAP-011; `README.md` line 79 link text updated. Current documentation = `README.md`, `DEPLOYMENT.md`, `doc/README.md`, `doc/leboncoin-session.md`, `doc/FUNCTIONAL_SPECIFICATIONS.md`, `doc/use-cases/`, `API_SPECIFICATION.md`.

TS-LBC-RM-004: create

- `doc/deprecated/README.md`: purpose of the folder (trace of removed functionality; history, not instructions) and a table: feature, removed on, replaced by, record link — first row “File-based LeBoncoin session (`LEBONCOIN_SESSION_FILE`)”, 2026-10-03 (date of the delivering commit), Settings › LeBonCoin session.
- `doc/deprecated/leboncoin-session-file.md` with sections: *What it was* (desktop capture to a private 0600 JSON file under a 0700 directory, secure `scp` transfer to the Raspberry Pi, installation as `/var/lib/pricefollower/leboncoin/session.json`, opt-in systemd drop-in `leboncoin-session.conf` setting `LEBONCOIN_SESSION_FILE`, collector-owned `session.json.state.json` sidecar bound to the import's SHA-256 fingerprint for automatic cookie renewal/revocation, renewal without restart); *Why removed* (user request of 2026-10-03, quoted: store the session in the database and set it from the interface so no file is required and the application is autonomous); *Replacement* (links to the settings, header menu, collection rev. 2 and capture rev. 2 specifications, the operator guide `../leboncoin-session.md`, and `API_SPECIFICATION.md`); *Clean-up of an existing deployment* (`sudo rm /etc/systemd/system/pricefollower.service.d/leboncoin-session.conf`, `sudo systemctl daemon-reload`, `sudo systemctl restart pricefollower`, then remove `/var/lib/pricefollower/leboncoin/` including `session.json`, `session.json.state.json` and any remote staging directory, then capture a fresh session and paste it in Settings; note that renewals stored in the sidecar are not imported); *History* (commit messages “feat(leboncoin): reuse a manually verified session in the collector”, “feat(leboncoin): add desktop session capture helper and operator guide”, “docs(leboncoin): record session specifications, review, QA and decisions”, “docs(leboncoin): replace trailing-space line breaks in session records”, how to find them with `git log --oneline --grep='leboncoin'` or `git log --grep='<message>'`, links to `../changes/2026-10-03-1213-leboncoin-session/index.md`, the superseded rows in the two revised functional specifications and the revision 1 sections of the two technical specifications; and the message(s) of the removal commit(s), filled by the coordinator at commit time without inserting a hash).
- `doc/README.md`: add the line `- [Deprecated functionality](deprecated/README.md) records removed features and their replacements.`

The drop-in path `/etc/systemd/system/pricefollower.service.d/leboncoin-session.conf` and session directory `/var/lib/pricefollower/leboncoin` were checked against the current revision 1 guide (`doc/leboncoin-session.md`, installation and drop-in sections) before its rewrite.

## Scope and refactoring

Only the files listed. History directories and superseded specification text are not edited (RM-006). No refactoring; deletion of the old mechanism is the requested functionality. `.tmp-session-qa/` (untracked, pre-existing) is not touched.

## Verification and unresolved questions

- `go build ./...`, `go test ./...`, `go test -race ./...`, `node --test scripts/capture-leboncoin-session.test.mjs`, `npm run build`, `npm run test:platform-tabs` (RM-007), `scripts/build-release.sh 6`.
- Search (RM-004), expected to match only `doc/deprecated/`, `doc/changes/2026-10-03-1213-leboncoin-session*/`, `doc/changes/2026-10-03-0854-leboncoin-403-investigation/` and superseded/history parts of `doc/specifications/leboncoin-session-*`:

  ```bash
  git grep -n -E 'LEBONCOIN_SESSION_FILE|session\.json|state\.json|leboncoin-session\.conf|--output' -- ':!doc/deprecated' ':!doc/changes' ':!doc/specifications'
  ```

  must return only the helper's `--output` rejection (TS-LBC-CAP-006) in `scripts/capture-leboncoin-session.mjs` and its test; nothing in application source, other scripts, README, DEPLOYMENT or the operator guide.
- Start the binary with `LEBONCOIN_SESSION_FILE` unset, set to an existing file and set to a missing path: identical startup; the settings endpoint reports `status: none` on a fresh database.
- `git diff --stat -- doc/changes/2026-10-03-1213-leboncoin-session doc/changes/2026-10-03-0854-leboncoin-403-investigation` is empty.
- All links in the two new files resolve (relative path check).

No unresolved question.

## Revision note

2026-10-03, review amendment REV-SET-001: status updated to record the API approval of 2026-10-03.
