# Issue #202 Clarify max-width override in the login form rule

**Problem:** `src/index.css:458-462` applies `max-width: none` to both `.login-form` and `.login-form .cds--btn[type="submit"]`, but only the button has a max-width to lift (Carbon's 20rem button cap).

**Suggested work:** Split the rule so `max-width: none` targets only the submit button, or add a short comment saying it lifts Carbon's button cap.

From review comment R1-04 on #198 (`doc/todo/issue-165-r1-04.md`).

## Comments by trusted authors

