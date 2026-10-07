# PR review round 2: issue #56

Re-review of the round-1 blocking fixes. No possible bugs found; all five round-1 blocking findings are resolved.

| PR | Reviewed head SHA | Event | Review |
|---|---|---|---|
| #87 | a0cd8a55939ca4b5e64d3f5a05c7d0042f751e6b | COMMENT | https://github.com/redjhawk/pricetracker/pull/87#pullrequestreview-5446334949 |
| #88 | 5712e7b5d0dcc8d4f8b9b79b2b6147944042b930 | COMMENT | https://github.com/redjhawk/pricetracker/pull/88#pullrequestreview-5446335132 |
| #89 | c7e65e0e5e123cccb589a83da19c79110a099fb8 | COMMENT | https://github.com/redjhawk/pricetracker/pull/89#pullrequestreview-5446335510 |
| #90 | 15b6420c50b81d6345dabcb2de6353ff96ef0db0 | COMMENT | https://github.com/redjhawk/pricetracker/pull/90#pullrequestreview-5446335740 |
| #91 | e1d597e2e4d3d78731cc40ad6e2236cae21b3b04 | COMMENT | https://github.com/redjhawk/pricetracker/pull/91#pullrequestreview-5446336114 |
| #92 | 5299b2d51e00b859cfad53d491205d062e369dac | COMMENT | https://github.com/redjhawk/pricetracker/pull/92#pullrequestreview-5446336397 |
| #97 | b21c688a5709bca79ff7c5de0b27120147f4f4e4 | COMMENT | https://github.com/redjhawk/pricetracker/pull/97#pullrequestreview-5446336575 |

## Comments

- #87: none. Finding 1 (cancelled request counted as a failure) resolved.
- #88: `internal/service/amazon_gate_test.go:79`, [test]: the shortened test starts from 0 failures, so an implementation that resets the count on cancel would still pass. Restore a non-zero precondition.
- #89: none. Findings 2 (busy loop) and 3 (failed save treated as success) resolved.
- #90: `internal/service/search_worker_test.go:138`, [test]: the follow-up commit removed the check that a later pass captures the items after a failed capture. Keep it.
- #91: none. Finding 4 (double delete) resolved.
- #92: none. Finding 5 (Amazon listing link) resolved.
- #97: none (documentation only).
