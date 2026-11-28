# Questions for issue #336

Context: the issue asks to extend the doc comment of `abortLateBodies` in `internal/httpapi/http_server.go` (lines 38-39) so it says the request body must be read on the handler's goroutine. That code does not exist on `master`. It was written for issue #273 (HTTP server read and write timeouts, PR #306) in commit `c87ff1a` on branch `ai-dev/issue-273`. That branch was later reverted to the tests stage and stopped there (`test-problem`: its locked test `internal/httpapi/http_server_limits_test.go` no longer compiles after master was merged in). So `internal/httpapi/http_server.go`, `abortLateBodies`, `lateBody` and the `http-server-timeouts` specification exist on neither `master` nor the tip of `ai-dev/issue-273`. A requirement on that comment can't be implemented or checked on this branch.

1. How should issue #336 go ahead, given that the code it documents is not merged?
   - A. Put #336 on hold until issue #273 is merged into `master`, then run it as a maintenance change on the merged code (a doc comment change only, no behaviour change).
   - B. Fold the request into issue #273: record it as a requirement of the `http-server-timeouts` specification on `ai-dev/issue-273`, so the #273 implementation writes the comment from the start, and close #336.
   - C. Close #336 as not applicable now, and reopen or refile it if `abortLateBodies` reaches `master`.
   - D. Other (please describe).
