# Implementation handoff

Status: implemented; source frozen for independent review. QA remains required.

Two distinct developers implement the ready capture and collection subjects with disjoint file ownership. Both recorded behavioral RED before implementing their feature. Their detailed commands, requirement mappings and outcomes are in [backend evidence](implementation-backend.md) and [capture evidence](implementation-capture.md).

Final developer checks passed: `go test ./...`, `go test -race ./...`, and all 13 capture helper tests through `node --test`. The capture helper also passed syntax and CLI help checks. Later corrections and their rechecks are recorded in the independent review and decisions.

The application API, frontend, SQLite schema, scheduling policy and dependencies are unchanged. No unrelated refactoring was performed. Session-assisted collection is opt-in through `LEBONCOIN_SESSION_FILE`; the desktop helper performs no automatic challenge interaction.

## Coordinator build verification

Executed from the repository on 2026-10-03:

```bash
PATH=/tmp/pricefollower-node22/node_modules/node/bin:$PATH GOCACHE=/tmp/pricefollower-session-go-cache scripts/build-release.sh 6
file release/pricefollower
go version -m release/pricefollower
```

The existing production build passed, including TypeScript and Vite. `file` reported a statically linked 32-bit ARM EABI5 ELF executable; Go build metadata reported Go 1.25.0, `GOOS=linux`, `GOARCH=arm`, `GOARM=6`, and `CGO_ENABLED=0`. The read-only module-cache metadata warning did not prevent successful compilation. The generated release is ignored by Git.

This verifies cross-compilation with the embedded frontend, not execution on a Raspberry Pi. No remote host was supplied and no remote deployment was attempted. Desktop capture and actual server acceptance require separate QA evidence.
