# Functional specification: ARMv6 remote deployment

Status: ready
Owner: independent functional specification agent
User reference: request to build for ARMv6, transfer and install on a remote device using existing scripts, with `pricefollower` as the default remote storage directory.

## Subjects, purpose and scope

One subject: run the existing release, transfer and installation workflow through one command. The actor is an operator with a configured SSH destination and permission to install the system service. Deployment includes the embedded frontend executable and installer, as already described in [DEPLOYMENT.md](../../../DEPLOYMENT.md); it does not transfer the source project or development data.

The SSH destination is necessary because no device address was supplied. The only optional deployment setting is the remote staging directory. Existing installation destinations and service settings remain those of the existing installer. No frontend, backend business behavior, HTTP API, refactoring, rollback system or device discovery is requested.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-ARMV6-001 | Operator has installed project dependencies and local Go, npm, SSH and rsync tools. | Provide one command accepting the SSH target and optional remote staging directory. | `./scripts/deploy-armv6.sh user@host [remote-directory]` is documented and works independently of the current local directory. Missing target, extra arguments and empty input fail with usage before deployment starts. |
| FR-ARMV6-002 | Valid invocation. | Build an ARMv6 Linux release using the existing release script. | The existing build script receives architecture `6`; the transferred executable is its newly built output with frontend assets embedded. A build failure prevents transfer and installation. |
| FR-ARMV6-003 | Build succeeds. | Create the remote staging directory if needed and transfer the binary and installer using the existing transfer script. | Omitted directory uses `pricefollower` relative to the SSH user's home. Explicit absolute and home-relative directories are supported. Existing directories work on repeated runs. Directory creation or transfer failure prevents installation. |
| FR-ARMV6-004 | Upload succeeds and remote device meets existing installer prerequisites. | Invoke the existing installer on the remote device with installation privileges. | Installation runs remotely, placing the executable in `/opt/pricefollower`, enabling and starting or restarting the service. Interactive sudo authentication works through an SSH terminal; unattended execution requires preconfigured authentication and installation privileges and must not receive stored credentials from the wrapper. |
| FR-ARMV6-005 | Deployment succeeds or a stage fails. | Report stages and propagate failures. | The command exits nonzero on validation, build, SSH, transfer, installer or final inactive-service failure. Success is reported only after `pricefollower` is active according to systemd. No rollback or continuous health monitoring is promised. |
| FR-ARMV6-006 | Existing installation contains tracked items. | Preserve the existing installer’s separation of staged release, installed executable and database. | The wrapper neither uploads a database nor deletes `/var/lib/pricefollower`; reinstalling retains persistent SQLite data under existing installer behavior. |
| FR-ARMV6-007 | Target or directory is operator-supplied. | Treat inputs as data, never executable shell instructions. | Accepted plain staging paths remain one literal path. Unsupported spaces, shell metacharacters and other target/path forms are rejected clearly before commands run, following the documented technical allowlist. Target values cannot inject SSH command options. The staged binary path must not collide with its parent staging directory. |

## States and corner cases

- Progress is a command-line stage report; no application loading or empty state changes are involved.
- Local dependencies are checked or errors propagated before the stage that requires them. Remote prerequisites remain Linux ARMv6 compatibility, SSH access, rsync, Bash, systemd and installer privileges.
- Relative directories are resolved from the remote SSH user's home; `pricefollower` is a staging directory, not a configurable installation or data location. Literal `~` expansion is not required; use a home-relative path or an absolute path.
- Whitespace-only input, newline/control characters, ambiguous SSH target syntax and unsupported path forms fail with actionable errors. The implementation documents its precise accepted syntax.
- Repeated runs rebuild and replace the staged release, then use the existing installer update behavior. Concurrent deployment, atomic upgrades, rollback, network retries and cleanup of staged release files are outside this request.
- The requested deliverable is the script. No actual device deployment is authorized or claimed during development without an SSH destination. QA must distinguish a mocked remote workflow from an executed real-device installation.
- Keyboard accessibility is normal command-line operation. Browser and API behavior are unchanged; no browser test is required for this deployment-only subject.

## Open questions

None affecting the script handoff. A real device address is needed only to execute a real deployment; the script requires it at invocation rather than inventing one.

## Traceability

- Existing behavior: [deployment instructions](../../../DEPLOYMENT.md), `scripts/build-release.sh`, `scripts/copy-dist.sh`, `scripts/install-pricefollower.sh`.
- Technical handoff: [technical specification](technical.md).
- Workflow evidence: [functional stage](../../changes/armv6-remote-deployment/functional-step.md).
