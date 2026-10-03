# Technical specification: ARMv6 remote deployment

Status: ready. Subject: one-command ARMv6 release deployment. Source: [functional requirements](functional.md).

## TS-ARMV6-001 — Scope and unaffected tiers

Add `scripts/deploy-armv6.sh` and document its usage in `DEPLOYMENT.md`. Preserve `build-release.sh`, `copy-dist.sh`, and `install-pricefollower.sh`; invoke their existing responsibilities instead of duplicating them. Workflow reports and subject specifications are the only other intended files. No refactoring is required.

Frontend: not affected; the existing release build compiles and embeds it without changing components, Carbon styling, routes, accessibility or client state. Backend: no Go, SQLite, configuration, scheduler, collector or persistence change. Existing installer behavior remains authoritative. API: not affected; [API_SPECIFICATION.md](../../../API_SPECIFICATION.md) remains the approved contract, with no methods, bodies, errors or semantics added. The command-line wrapper is not an HTTP API. Maps FR-001 through FR-006.

## TS-ARMV6-002 — Command interface and validation

Use Bash and `set -euo pipefail`, resolving the project directory from `BASH_SOURCE` in the existing script style. Invocation: `./scripts/deploy-armv6.sh user@host [remote-directory]`. Accept one or two arguments; omitted directory defaults to `pricefollower`. Explicit empty inputs and unsupported syntax exit nonzero with an actionable usage error before building or contacting a host.

Accept an ASCII host or SSH alias composed of letters, digits, dots, underscores and hyphens, beginning with a letter or digit; optionally prefix one username and `@`, with username beginning with a letter, digit or underscore and otherwise containing letters, digits, underscores, dots or hyphens. Reject options beginning with `-`, multiple `@`, colons, whitespace, control characters and shell syntax. IPv6 literals and inline ports are outside supported syntax; an SSH config alias can supply those connection settings. Pass the validated destination as one quoted argument to SSH.

Accept staging paths using only ASCII letters, digits, underscores, dots, hyphens and `/`, beginning with a letter, digit, underscore, dot or `/`. Strip trailing slashes for consistent use; reject an empty result, `/`, whole `.`, and any `..` path segment. Absolute and home-relative paths are supported, including `./pricefollower`; literal `~`, spaces and shell metacharacters are rejected clearly. Quote accepted paths in remote shell commands even with validation, use `--` in directory operations, and append `/` for the transfer destination. Reject absolute staging paths equal to `/opt/pricefollower` or `/var/lib/pricefollower`, and their descendants, to avoid writing directly into the live installation or persistent data. For these checks, use a separate lexical check value: collapse repeated `/` and `/./` separators and remove trailing `/.`; also reject normalized root, empty or whole-dot values so `/./` cannot bypass root rejection. Keep the original validated staging path unchanged for remote commands and transfer. This small normalization is necessary because otherwise `/opt//pricefollower` and `/opt/./pricefollower` bypass protection of the same directory. This is a simple syntactic safeguard, not remote path canonicalization: operators remain responsible for choosing an ordinary staging directory without symlink aliases into live installation or data. No `eval` or credential storage. Document the allowlist instead of implying arbitrary paths work. Maps FR-001, FR-003, FR-007.

## TS-ARMV6-003 — Sequential stages

Check local `go`, `npm`, `ssh` and `rsync` availability before the build. Existing project dependencies must already be installed; retain their existing build failure behavior. Print concise stage progress and perform these steps in order:

1. Invoke the existing build script with argument `6` from the resolved project directory. It produces `release/pricefollower` for `GOOS=linux`, `GOARCH=arm`, `GOARM=6`, `CGO_ENABLED=0`, embedding the frontend.
2. SSH to the validated target to create the staging directory with `mkdir -p -- 'validated-path'`. Relative paths resolve from the remote login user's home. Failure stops processing.
3. Invoke `copy-dist.sh` with one destination argument `target:validated-path/`. Existing rsync transfers the newly built binary and installer; it does not transfer source files or a SQLite database. Failure stops processing.
4. Invoke SSH with `-t` to permit interactive sudo: `cd -- 'validated-path' && sudo bash ./install-pricefollower.sh ./pricefollower && systemctl is-active --quiet pricefollower`. All remote commands must be conditional so an earlier failure prevents subsequent operations. Explicit Bash invocation avoids relying on transferred executable permission. Require remote sudo even for root SSH accounts; document that prerequisite rather than adding a privilege detection mechanism.
5. Print success only when the remote command completes successfully, including the final service-active check. Propagate nonzero failures without rollback, retries or unconditional success output.

The staging directory is a directory containing the child file `pricefollower`; default `pricefollower` therefore resolves to `~/pricefollower/pricefollower`, not the parent directory as a binary. Operators should use a separate staging directory rather than the application or data directory. Existing installation remains `/opt/pricefollower`, data `/var/lib/pricefollower`, and service `pricefollower` on port 3001. Remote prerequisites: ARMv6-compatible Linux, Bash, rsync, sudo, SSH and systemd plus the existing installer utilities and privileges. SSH and sudo authentication remain ordinary operator configuration. Maps FR-002 through FR-007.

## TS-ARMV6-004 — Verification and limitations

Run Bash syntax validation for the wrapper and existing called scripts. Build a real ARMv6 release when available dependencies permit it; distinguish successful real cross-compilation from mocked orchestration.

Use isolated temporary fixtures with stub build/transfer commands and fake SSH binaries to execute the command-line interface without contacting a device. Record ordered invocations and validate default directory, explicit relative/absolute directories, operation from another working directory, repeated calls, final service-active gating, failures at each stage, dependency failure and rejected malformed/injection inputs. Remote shell commands can additionally be executed in an isolated fixture with fake sudo/systemctl/installer to verify directory selection and conditional execution. Do not replace production scripts to test them. Record deterministic exploratory inputs and expectations.

No actual device destination was provided; real transfer, sudo prompting, installed service activity, reboot behavior and existing-database preservation remain unexecuted device checks, not passed results. Browser URLs are not applicable to this command-line change. QA records actual commands and fixture locations; no invented application URL. No external library, API approval or refactoring gate is required.
