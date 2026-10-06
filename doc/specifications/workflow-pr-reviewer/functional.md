# Workflow PR reviewer: functional specification

Status: ready
Source: user request of 2026-10-06, with answers to clarifying questions: GitHub PR review comments; comment only (the user decides fixes); for possible bugs, submit "Request changes" and inform the user on the issue or in the chat.
Scope: development workflow, a new agent role, and the ai-dev prompt. Application behavior is not affected.

## Requirements

- **FR-WORKFLOW-PRR-001** After the pull requests are created, a new, separate agent reviews each pull request.
- **FR-WORKFLOW-PRR-002** The agent is an expert coder in Go (backend) and JavaScript (frontend), and reviews every detail of the diff.
- **FR-WORKFLOW-PRR-003** It comments on everything that could be improved, applying these rules: refactoring goes in a dedicated PR; code is readable, not clever and unreadable; when there is a simpler way, the simplest way is used.
- **FR-WORKFLOW-PRR-004** It must catch possible bugs.
- **FR-WORKFLOW-PRR-005** Comments are posted as a GitHub pull request review, inline on the diff.
- **FR-WORKFLOW-PRR-006** The agent only comments. The user decides which comments are fixed.
- **FR-WORKFLOW-PRR-007** When a possible bug is found, the review is submitted as "Request changes", and the user is informed on the originating issue or in the chat interface.

## Acceptance criteria

- A role file defines the agent, and `WORKFLOW.md` and `AGENTS.md` include it as a stage after pull-request creation.
- The ai-dev run is allowed to post PR reviews, and its prompt runs the agent after opening the PRs and does not fix its comments unprompted.
