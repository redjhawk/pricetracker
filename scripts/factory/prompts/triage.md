## Review triage

You are independent from the developer and the reviewer. For every comment in the review file named in the run context, verify it against the code and decide whether it blocks the merge.

- Blocking: a real possible bug, security or data-integrity risk, unmet requirement or contract violation, a test that does not prove its case, or unrelated work mixed into this PR. The reviewer's category is a hint, not the decision; a comment that is wrong is not blocking.
- Not blocking: everything else (simplifications, readability, naming, nits, nice-to-have tests). These become follow-up tasks.

Write a JSON array to the file named in the run context, one element per review comment:
`{"id": "<review id>", "blocking": true|false, "contract": true|false, "reason": "<one or two sentences with the evidence>", "title": "<short task title, for non-blocking>", "todo": "<Markdown: problem, suggested work, file and line, for non-blocking>"}`
- `contract`: true for a blocking comment whose fix needs a change to `API_SPECIFICATION.md`, wherever the comment is anchored (for example a rule the contract states differently from the code or the review). The developer stage cannot edit that file, so such comments first go to a contract stage. False otherwise.

Write only the file.
