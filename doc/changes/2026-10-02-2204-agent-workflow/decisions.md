# Review decisions: agent workflow

Reviewed requirements: the user's specification-led team workflow request and [repository instructions](../../../AGENTS.md).
Reviewed documentation: [workflow](../../workflow/WORKFLOW.md), its templates, and the six [role instructions](../../../.agents/roles/).
Reviewed revision: working-tree documentation snapshot; final independent recheck is recorded in [review.md](review.md).
Reviewer: `workflow_reviewer`, independent of implementation.
Adjudicator: `workflow_adjudicator`, distinct from reviewer and documentation authors.

## Findings and decisions

### REV-001: Review finding ID differs between workflow and template

- Evidence: the reviewer reported a template heading using `RV-01` while the workflow specifies `REV-001`.
- Impact: copying the template would create inconsistent IDs and make review-to-decision traceability less reliable.
- Criticality: noncritical. This does not change application behavior or remove a required approval, and the finding remains identifiable; it is a documentation consistency defect.
- Disposition: fix.
- Reason: aligning the example is a narrow correction that prevents recurring inconsistency without adding process or changing user requirements.
- Specification decision: not required; the workflow already defines the ID convention.
- Resolution: fixed; template uses `REV-001`, independently verified in [REV-001 recheck](review.md#rev-001-review-identifier-prefix-differs).
- Owner/follow-up: coordinator; reviewer confirmation recorded above, no remaining action.

### REV-002: Template mixes pending user decisions with dispositions

- Evidence: the template previously allowed pending user decisions within a disposition whose documented values are `fix`, `defer`, and `reject`.
- Impact: a record could mistake an unresolved user choice for a final decision, or use an inconsistent disposition across reports.
- Criticality: noncritical for this documentation change. No approval was fabricated and no application implementation relies on the example, but the ambiguity merits correction before the workflow is used.
- Disposition: fix.
- Reason: keep disposition unset when a required user decision is pending and represent that condition in resolution. This preserves the workflow's approval gates and its existing vocabulary.
- Specification decision: not required; no product choice is being made.
- Resolution: fixed; pending status is separate from disposition, independently verified in [REV-002 recheck](review.md#rev-002-pending-user-decisions-are-mixed-with-final-dispositions).
- Owner/follow-up: coordinator; reviewer confirmation recorded above, no remaining action.

### REV-003: Feature index has no specified artifact path

- Evidence: the workflow required a feature index without giving its location.
- Impact: separate agents could create different index files, obscuring stage status, handoffs, and links.
- Criticality: noncritical. The required index content and staged process are already clear; the defect concerns discoverability rather than missing behavior or an unsafe authorization.
- Disposition: fix.
- Reason: `doc/changes/<change>/index.md` fits the established report directory and gives agents one predictable handoff location without introducing additional artifacts.
- Specification decision: not required; this is a routine documentation organization choice.
- Resolution: fixed; explicit index path added and independently verified in [REV-003 recheck](review.md#rev-003-feature-index-location-is-unspecified).
- Owner/follow-up: coordinator; reviewer confirmation recorded above, no remaining action.

### REV-004: Roles impose universal specification approval

- Evidence: functional, technical, and development role instructions referred to approved functional/technical files even when the canonical workflow permits ready files and requires no repeated confirmation of explicit user requirements.
- Impact: agents could add redundant user signoffs and block a handoff already authorized by clear requirements. This conflicts with the intended staged workflow and the user's preference for autonomous progress within approved scope.
- Criticality: noncritical for the documentation-only delivery. The defect would delay work rather than bypass an approval or corrupt application data, and no feature implementation was blocked or performed under it.
- Disposition: fix.
- Reason: use ready specifications plus explicit approval where the workflow requires it. Preserve mandatory user decisions for product ambiguities, API contract changes, and refactoring. Removing blanket wording corrects the extra gate without weakening those controls.
- Specification decision: not required; the existing instructions establish the relevant gates.
- Resolution: fixed; all six roles independently rechecked for ready specifications and required decisions in [REV-004 recheck](review.md#rev-004-role-prompts-imply-additional-specification-approval-gates).
- Owner/follow-up: role author and reviewer; all affected handoffs verified, no remaining action.

### REV-005: Adjudicator role adds a conflicting disposition value

- Evidence: the adjudicator role listed `seek user decision` alongside `fix/defer/reject`, contrary to the workflow and decision template.
- Impact: agents could produce incompatible records or blur the difference between an outstanding user choice and an adjudicated outcome.
- Criticality: noncritical for this documentation-only change. The same role explicitly prohibits approving user choices, so the inconsistency is not evidence of a waived gate; it still undermines reliable records.
- Disposition: fix.
- Reason: retain the three final dispositions and track pending user decisions separately. This is the same rule applied in REV-002 and keeps role instructions compatible with their template.
- Specification decision: not required; the canonical workflow already defines dispositions.
- Resolution: fixed; disposition and pending status separated, independently verified in [REV-005 recheck](review.md#rev-005-adjudicator-role-uses-a-pending-decision-as-final-disposition).
- Owner/follow-up: role author and reviewer; enum and pending-state wording verified, no remaining action.

## Readiness

All five findings warrant narrow fixes. None is deferred or rejected, so no comment is intentionally left unfixed. No application code, API contract, or product behavior changes are adjudicated here. All five corrections are independently verified in the review report; no unresolved review blocker remains. Final workflow completion awaits documentation QA. Application interface tests are not applicable to this documentation-only scope.
