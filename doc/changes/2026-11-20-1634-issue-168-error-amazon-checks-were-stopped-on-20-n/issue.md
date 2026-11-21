# Issue #168 Error. Amazon checks were stopped on 20 Nov 2026, 07:26 UTC after repeated failures.

I think Amazon has detected that you're requesting as a bot.
I suggest you to use a real navigator to request pages so it really simulates a request.
Moreover, you also always request the same hour each day. You must random a bit more your request as if you were a human. You have quite a big marge of hours to the the request so random the request time more



## Comments by trusted authors

### redjhawk at 2026-11-20T17:10:39Z

1. between 6 et 9 et 19-23h
2. no. amazon only. leboncoin is running fine now
3. c
4. b

### redjhawk at 2026-11-20T17:30:09Z

can you continue ? 

### redjhawk at 2026-11-20T17:45:06Z

continue 

### redjhawk at 2026-11-20T17:47:17Z

@claude  en la rama principal hay una corrección para el error que estás teniendo al desarrollar. Haz primero un rebase de main y sigue después desarrollando

### redjhawk at 2026-11-20T18:36:16Z

@claude  continue please

### redjhawk at 2026-11-20T19:48:04Z

1. between 6 et 9 et 19-23h
2. no. amazon only. leboncoin is running fine now
3. C
4.b 

### redjhawk at 2026-11-20T20:18:36Z

1. between 6 et 9 et 19-23h
2. no. amazon only. leboncoin is running fine now
3. C
4.b 

What don't you develop something?

@claude 

### redjhawk at 2026-11-20T21:33:12Z

ignore the test case and continue!!!

### redjhawk at 2026-11-20T22:01:25Z

testcases gate fixed by hand: the 4 requirements whose definition lines this change modifies (FR-AMZ-STOP-001/004/008, FR-AMZ-SRCH-DEL-006) are now mapped to existing test cases (22d09df). Stage advanced to `techspec`.

### redjhawk at 2026-11-21T10:26:40Z

@claude continua 

### redjhawk at 2026-11-21T12:32:31Z

There is a fix on master @claude 

### redjhawk at 2026-11-21T20:45:36Z

@claude  prueba de nuevo

### redjhawk at 2026-11-21T21:04:32Z

@claude try again 

### redjhawk at 2026-11-21T22:02:13Z

@claude Fixed on master (#178) and merged into `ai-dev/issue-168` (615b295):

- The responsive-layout 320px/360px tests failed because the Carbon spacing tokens were not declared on `:root`. They are now; `npx playwright test` passes 84/84 on this branch.
- `internal/service/settings.go` is gofmt-ed. The only remaining `gofmt -l .` entry is the locked test `internal/amazon/browser_test.go`; leave it as is, it is not a blocker for this stage.
- The README text from TS-AMZ-HCHK-012 can be added later in a separate PR; it is not a blocker.

Continue the implement stage.

