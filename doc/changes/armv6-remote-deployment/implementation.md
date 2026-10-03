# ARMv6 deployment implementation

Status: implemented; ready for independent review and command-line QA.
Role: expert developer. Source: [functional](../../specifications/armv6-remote-deployment/functional.md), [technical](../../specifications/armv6-remote-deployment/technical.md), [API preservation](api-step.md).

Read the developer role, workflow and all four required project skills. No contract change or refactoring is needed.

## Scoped implementation

Added executable `scripts/deploy-armv6.sh` accepting a validated SSH target and optional staging directory defaulting to `pricefollower`. It resolves its project directory independently of the caller's working directory, checks local tools, then calls unchanged `build-release.sh 6`, remote `mkdir`, unchanged `copy-dist.sh`, and the existing installer through `ssh -t` and `sudo bash`. Conditional remote commands and Bash failure propagation prevent continuation after errors; success requires systemd's active-service check.

ASCII allowlists prevent shell/SSH option injection. The protected absolute path guard collapses repeated separators and dot segments locally for comparison while preserving the actual literal staging path; it does not resolve remote symlinks. Parent segments, root/dot, unsupported characters and installed/data directories are rejected before building. Credentials are not stored; databases are not transferred or deleted by this wrapper. Updated `DEPLOYMENT.md` with invocation, precise syntax, prerequisites, persistence and limitations. Existing build, transfer and installer scripts are unchanged.

## Executed verification

| Command | Actual result |
| --- | --- |
| `bash -n scripts/deploy-armv6.sh scripts/build-release.sh scripts/copy-dist.sh scripts/install-pricefollower.sh` | Passed. |
| `./scripts/build-release.sh 6` | Passed, exit 0. TypeScript and Vite built 947 modules; Go produced the embedded frontend release. Go emitted a nonfatal read-only module stat-cache warning; compilation and output succeeded without escalation. |
| `file release/pricefollower` | ELF 32-bit ARM EABI5, statically linked, stripped executable. |
| `go version -m release/pricefollower` | Go 1.25.0 metadata confirms `GOOS=linux`, `GOARCH=arm`, `GOARM=6`, `CGO_ENABLED=0`. |
| `git diff --check` | Passed. |

These checks establish syntax and a real ARMv6 cross-build, not an executed deployment. No remote destination was supplied; no SSH connection, real transfer, sudo prompting, service installation or device/database persistence check was executed. Independent QA must test orchestration using isolated fixtures and report device checks separately. Browser QA is inapplicable to this CLI-only change. No commit, push or existing-script refactoring was performed by this role.
