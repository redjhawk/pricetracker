# Blocking review comments (round 2)

## R2-01 `API_SPECIFICATION.md:706`

The list of endpoints that read a request body leaves out `POST /api/v1/admin/users`. That endpoint reads its body through `decodeCredentials` (auth.go:162 → auth.go:214-216), so `abortLateBodies` covers it too. With a body sent one byte per second it is dropped at 15 s with no response and no user created. The row instead files it under 'Any other request is handled once its headers arrive ... its response is sent with `Connection: close`', which describes the opposite behaviour. The same omission is in FR-HTTP-TMO-001 (functional.md:26: 'login, adding an item, saving a purchase goal, saving settings'), FR-43 and technical.md:57. The code does the right thing, so the fix is documentation only: add `POST /api/v1/admin/users` (creating a user) to the list in this row, in FR-HTTP-TMO-001 and in technical.md:57.

Triage: Confirmed: POST /api/v1/admin/users decodes its body through decodeCredentials (auth.go:162, 213-216) behind abortLateBodies (server.go:32), so a slow body is dropped with no response. The API_SPECIFICATION.md row for the 15 s limit lists the body-reading endpoints without it and files it under 'Any other request is handled once its headers arrive', so the contract added by this PR states the opposite of the behaviour; FR-HTTP-TMO-001's list and technical.md have the same omission.

