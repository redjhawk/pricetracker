# Command-line QA: ARMv6 remote deployment

Status: executed; all 43 isolated cases passed, no new findings.
Tester: independent `documentation_qa` invocation acting as expert QA.
Date: 2026-10-03, Europe/Paris session date.
Revision: working-tree wrapper over HEAD `14e45629291b0a7b52fd233191e284a208a54e3e`, following corrected [review](review.md) and [adjudication](decisions.md).

## Executed environment and commands

Ran `python3 /tmp/armv6-qa.py` and `bash -n scripts/deploy-armv6.sh scripts/build-release.sh scripts/copy-dist.sh scripts/install-pricefollower.sh`; both exited 0. The Python command created an isolated fixture at `/tmp/pricefollower-armv6-qa-wu4vqg7_`, copied all four actual scripts unchanged, and invoked its wrapper with `/bin/bash` from `/tmp`, outside the fixture project. PATH stubs intercepted `npm`, `go`, `ssh`, and `rsync`, recording arguments, working directory and cross-build environment. The actual build/copy scripts executed their normal orchestration; the fake npm produced a temporary dist file and fake Go wrote a fixture release binary.

Detailed actual command arguments, outputs, exit codes and tool events are in `/tmp/pricefollower-armv6-qa-wu4vqg7_/results.json`; harness is `/tmp/armv6-qa.py`. These temporary artifacts may be removed later; the executed case catalog below retains the key expectations and outcomes. Test targets `pi@qa-device`, `qa-alias`, and `pi@host` were stub-only strings, never contacted hosts. No credentials, remote session, database changes or real system service mutations occurred.

## Executed case catalog

| Cases | Requirements / category | Actions / expected | Actual results |
| --- | --- | --- | --- |
| CLI-001 | FR-ARMV6-001,002,003,004,006; default/CWD-independent workflow | Run fixture wrapper `pi@qa-device` from `/tmp`; inspect ordered tools, ARM env, embedded fixture and upload inputs | Exit 0. Order `npm → go → ssh mkdir → rsync → ssh install`. Go received `GOOS=linux`, `GOARCH=arm`, `GOARM=6`, `CGO_ENABLED=0` and saw embedded `isolated frontend`. Default mkdir path `pricefollower`; rsync destination `pi@qa-device:pricefollower/`; exactly binary and existing installer supplied, no database/source project. Final SSH uses `-t` and conditional installer/service check. |
| CLI-002-0..3 | FR-001,003,007; explicit path forms | Run `qa-alias` with `custom/stage`, `/tmp/qa-stage`, `./pricefollower`, `stage///` | Four exits 0 with the same ordered stages. Absolute, home-relative and dot-relative inputs accepted; trailing separators stripped for commands/destination. Paths remain quoted single remote operands. |
| CLI-003-repeat | FR-003; repeated deployment | Run default invocation a second time in the same fixture | Exit 0; rebuild and complete ordered stages again. Local existing release/staging preparation causes no collision. Actual remote existing-directory handling remains mkdir-p intent, not a device test. |
| CLI-004-npm/go/mkdir/rsync/install/inactive | FR-002,003,005; stage failures | Set one controlled failing stage to exit 17 per run | Six exits 17; exact tool prefix ends at failing stage, no later tools and no “Deployment complete” message. Inactive service modeled as final remote command failure. |
| CLI-005-0..23 | FR-001,007; invalid input/injection/protected destinations | Execute 24 argument sets below | All exit 2 before any stub tool runs; no success message. |
| CLI-006-go/npm/ssh/rsync | FR-001,005; absent local dependency | Construct four dedicated PATHs with one dependency missing; invoke wrapper | Four exits 1; no build/SSH/transfer begins. |
| remote-success | FR-003,004,005; captured remote command execution | Evaluate actual captured install command with Bash from isolated mock home; fake sudo executes temporary fixture installer; fake systemctl succeeds | Exit 0; fixture installer sees `/tmp/pricefollower-armv6-qa-wu4vqg7_/mock-home/pricefollower` and binary argument `./pricefollower`; then `systemctl is-active --quiet pricefollower`. No real installer/service execution. |
| remote-install-failure | FR-004,005; conditional remote sequence | Fake fixture installer exits 9 | Exit 9; installer event only, systemctl does not run. |
| remote-inactive | FR-005; final service gate | Fake fixture installer succeeds, fake systemctl exits 9 | Exit 9; installer and service check run, confirming failed final gate remains nonzero. |

## Exact malformed-input sequence

Deterministic ordered cases; no random generator or seed used. Values are shown as JSON argument arrays so empty strings/newlines and shell characters are unambiguous data.

```json
[
  [],
  [
    ""
  ],
  [
    "   "
  ],
  [
    "pi@host",
    "stage",
    "extra"
  ],
  [
    "-oProxyCommand=bad"
  ],
  [
    "pi@host;touch_bad"
  ],
  [
    "a@@host"
  ],
  [
    "host:22"
  ],
  [
    "::1"
  ],
  [
    "host\ncmd"
  ],
  [
    "pi@host",
    ""
  ],
  [
    "pi@host",
    " "
  ],
  [
    "pi@host",
    "/"
  ],
  [
    "pi@host",
    "."
  ],
  [
    "pi@host",
    "a/../b"
  ],
  [
    "pi@host",
    "../stage"
  ],
  [
    "pi@host",
    "~/.stage"
  ],
  [
    "pi@host",
    "a b"
  ],
  [
    "pi@host",
    "a;cmd"
  ],
  [
    "pi@host",
    "$(cmd)"
  ],
  [
    "pi@host",
    "/opt/pricefollower"
  ],
  [
    "pi@host",
    "/opt//pricefollower/a"
  ],
  [
    "pi@host",
    "/opt/./pricefollower/a"
  ],
  [
    "pi@host",
    "/var/lib/pricefollower/db"
  ]
]
```

## Coverage and limits

This is executed CLI fixture QA, not an executed deployment or genuine cross-compilation. The developer's real ARMv6 build and reviewer's independent Go binary metadata verification are recorded separately in [implementation](implementation.md) and [review](review.md). Fixture checks establish wrapper orchestration, quoting/validation, conditional remote command behavior and failure propagation with controlled dependencies.

No real device destination was supplied. Actual SSH transfer, network authentication, interactive sudo prompting, live installer execution, persistent database preservation, systemd activity, ARM device startup and reboot remain unexecuted device checks. Database preservation is supported by unchanged installer review and the observed binary/installer-only upload inputs, not a tested real database upgrade. Symlink aliases into protected directories, concurrent deployment, rollback, retry and remote canonicalization are documented outside scope. No application behavior changed, so browser interface URLs/random browser tests are not applicable and none were invented.

Production scripts were not replaced; fixture copies and stub tools stayed under `/tmp`. Temporary mock data has no production effect and may be deleted with its fixture directory. No new findings or unresolved QA blocker; coordinator may proceed to focused staged checks and commit, preserving the real-device limitation in the final handoff.
