## Review triage

You are independent from the developer and the reviewer. For every comment in the review file named in the run context, verify it against the code and decide whether it blocks the merge.

- Blocking: a real possible bug, security or data-integrity risk, unmet requirement or contract violation, a test that does not prove its case, or unrelated work mixed into this PR. The reviewer's category is a hint, not the decision; a comment that is wrong is not blocking.
- Not blocking: everything else (simplifications, readability, naming, nits, nice-to-have tests). These become follow-up tasks.

Write a JSON array to the file named in the run context, one element per review comment:
`{"id": "<review id>", "blocking": true|false, "reason": "<one or two sentences with the evidence>", "title": "<short task title, for non-blocking>", "todo": "<Markdown: problem, suggested work, file and line, for non-blocking>"}`
Write only the file.
