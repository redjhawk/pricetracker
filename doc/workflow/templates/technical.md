# Technical specification: <subject>

Status: draft | needs-clarification | ready | superseded
Functional specification: <link and revision>

## Requirement mapping

| Functional ID | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- |
| <ID> | <simplest sufficient design> | <paths> | <observable check> |

## Frontend

<Components, state, accessibility, navigation, loading/errors. Explicitly state “no change” and why when unaffected.>

## Backend

<Handlers, service rules, persistence, collection, validation, cancellation, migration and data preservation as applicable. State “no change” and why when unaffected.>

## API

<Methods/paths, request/response fields and nullability, statuses, safe errors, asynchronous semantics, compatibility. Reference API_SPECIFICATION.md; state no contract change when appropriate.>

## Scope and refactoring

<Exact change boundary, dependencies, risks. For each proposed refactor, explain need, impact, and before/after options; record the decision and rationale before doing it.>

## Verification and unresolved questions

<Build checks and functional/QA cases mapped to requirements; unresolved decisions block dependent work.>
