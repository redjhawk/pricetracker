# ARMv6 remote deployment script

Stage: commit prepared. Real ARMv6 build and 43 isolated CLI QA scenarios passed; review finding fixed and independently rechecked. Git outcome is reported in final handoff.
User source: create a script to build for ARMv6, transfer the release and install remotely by reusing existing scripts; default remote staging directory `pricefollower`.

## Results

1. [Functional specification](../../specifications/armv6-remote-deployment/functional.md) and [handoff](functional-step.md).
2. [Technical specification](../../specifications/armv6-remote-deployment/technical.md) and [handoff](technical-step.md).
3. [API assessment](api-step.md).
4. [Implementation](implementation.md).
5. [Independent review](review.md).
6. [Independent decisions](decisions.md).
7. [QA](qa.md).
8. [Commit handoff](commit-step.md).

## Decisions and scope

The command needs an SSH destination identifying the device. The optional path defaults to `pricefollower` relative to the remote user's home. This request creates the script; no actual device or credentials were supplied, so no real remote installation is performed. Application tiers and the approved API are unaffected. Existing build/copy/install scripts remain the source of release/install behavior.

Agents: `arm_deploy_functional` functional specification; `arm_deploy_technical` technical/API specification; `marketplace_developer` implementation. Reviewer, adjudicator and QA remain independent of the implementation.

Independent review: `workflow_adjudicator` acting as reviewer only. Independent adjudication: `/root`, distinct from script developer and reviewer. CLI QA: `documentation_qa`; browser testing is not applicable to this deployment-only change.
