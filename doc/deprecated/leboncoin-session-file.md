# File-based LeBoncoin session (removed)

Removed on 2026-10-03 by change [leboncoin-session-settings](../changes/2026-10-03-1741-leboncoin-session-settings/index.md). Replaced by **Settings › LeBonCoin session**.

## What it was

The file-based session let the collector reuse a LeBoncoin `datadome` cookie that the operator had verified manually in a desktop browser:

1. **Desktop capture to a private file.** `scripts/capture-leboncoin-session.mjs --url <listing> --output <file>` opened an isolated browser, waited for the operator to complete any LeBoncoin verification, and wrote the applicable `datadome` cookie with its metadata as a versioned JSON file (mode 0600) in a private directory (mode 0700) outside the repository, replacing an earlier export atomically.
2. **Secure transfer to the Raspberry Pi.** The operator copied the file with `scp` into a private staging directory on the Pi, then installed it as `/var/lib/pricefollower/leboncoin/session.json`, owned by the `pricefollower` service user in a mode-0700 directory, through an atomic rename.
3. **Opt-in systemd drop-in.** `/etc/systemd/system/pricefollower.service.d/leboncoin-session.conf` set `Environment=LEBONCOIN_SESSION_FILE=/var/lib/pricefollower/leboncoin/session.json`, followed by `systemctl daemon-reload` and a service restart.
4. **Collector-owned sidecar.** The collector kept `session.json.state.json` next to the import, bound to the import's SHA-256 fingerprint, to store renewed cookies that LeBoncoin returned on successful verified pages and to record revocation.
5. **Renewal without restart.** Installing a new `session.json` took effect on the next collection attempt.

## Why it was removed

User request of 2026-10-03:

> instead of using a file for storing the session, why not storing it in the db, and let the user to set it from the interface ? […] This way, no file is required and application is autonomous. Delete all references to old system.

The session is now stored in the application's database and set from the interface, so no file, transfer, environment variable, server shell access or restart is needed.

## What replaces it

- A profile icon at the top right of the header opens a menu with **Settings**; the settings modal has one entry, **LeBonCoin session** ([header menu specification](../specifications/app-header-menu/functional.md), [session settings specification](../specifications/leboncoin-session-settings/functional.md)).
- The session value is stored in SQLite; renewed cookies are still saved automatically, and the modal shows the last failure of the saved session ([collection specification, revision 2](../specifications/leboncoin-session-collection/functional.md#requirements-revision-2)).
- The capture helper prints a `datadome=<value>` line to paste into Settings instead of writing a file ([capture specification, revision 2](../specifications/leboncoin-session-capture/functional.md#requirements-revision-2)).
- Settings endpoints: [API specification](../../API_SPECIFICATION.md).
- Operator guide: [LeBoncoin session from Settings](../leboncoin-session.md).

## Clean-up of an existing deployment

The application no longer reads the environment variable, the session file or the sidecar, and it does not delete or import them. Renewals stored in the old sidecar are not imported. On the Raspberry Pi:

```bash
sudo rm /etc/systemd/system/pricefollower.service.d/leboncoin-session.conf
sudo systemctl daemon-reload
sudo systemctl restart pricefollower
sudo rm -r /var/lib/pricefollower/leboncoin
```

The last command removes `session.json`, `session.json.state.json` and their directory. Also remove any leftover `~/.pricefollower-session-transfer.*` staging directory in your SSH account on the Pi and the desktop export directory (for example `~/.pricefollower-session`). Then capture a fresh session and paste it into **Settings › LeBonCoin session** as described in the [operator guide](../leboncoin-session.md).

## History

The feature was delivered by these commits (find them with `git log --oneline --grep='leboncoin'` or `git log --grep='<message>' --fixed-strings`):

- “feat(leboncoin): reuse a manually verified session in the collector”
- “feat(leboncoin): add desktop session capture helper and operator guide”
- “docs(leboncoin): record session specifications, review, QA and decisions”
- “docs(leboncoin): replace trailing-space line breaks in session records”

Records and specifications:

- Change records: [leboncoin-session](../changes/2026-10-03-1213-leboncoin-session/index.md).
- Superseded requirements: [collection, revision 1](../specifications/leboncoin-session-collection/functional.md#requirements-revision-1) (FR-LBC-COL-001 to 006, 009, 010) and [capture, revision 1](../specifications/leboncoin-session-capture/functional.md#requirements-revision-1) (FR-LBC-CAP-001, 005 to 009).
- Revision 1 technical designs: [collection](../specifications/leboncoin-session-collection/technical.md#revision-1-technical-design-history) and [capture](../specifications/leboncoin-session-capture/technical.md#revision-1-technical-design-history).

Removal commits (change leboncoin-session-settings): `feat(leboncoin): store the session in SQLite behind a settings API` (removes `LEBONCOIN_SESSION_FILE` and the file/sidecar code) and `feat(leboncoin): print captured session for the Settings modal and deprecate the session file` (capture helper no longer writes files; operator guide and this record).
