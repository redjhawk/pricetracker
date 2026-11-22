# Blocking review comments (round 3)

## R3-01 `API_SPECIFICATION.md:688`

The API contract still says that `POST /api/v1/items/{id}/refresh` and `POST /api/v1/items/refresh` "no longer change `nextCheckAt` of Amazon items". After the round-2 fix (R2-01), RefreshItem calls rescheduleOverdueAmazonCheck (internal/service/service.go:304, 325-331). When the stored next check is already due (<= now), it stores nextAmazonCheckAt(now). A client can see this: refresh a stopped Amazon item whose nextCheckAt is in the past, then call GET /api/v1/items, and nextCheckAt has moved to the next window. TS-AMZ-HCHK-007 and the Schedule section of technical.md were updated for this, but API_SPECIFICATION.md was not, so the two documents disagree. Fix: reword the bullet, e.g. "... they no longer change a future `nextCheckAt` of Amazon items; Refresh price moves an already passed Amazon `nextCheckAt` (a check skipped while the item was stopped) to a random time in the next window (LeBoncoin unchanged)."

Triage: Confirmed contract violation. API_SPECIFICATION.md:688 says the two refresh endpoints "no longer change `nextCheckAt` of Amazon items". But RefreshItem (internal/service/service.go:302-304, 325-331) stores a new window time when the stored one is already due, and clients can see that in GET /api/v1/items. TS-AMZ-HCHK-007 and technical.md:106 describe the new behaviour; technical.md:143 has the same outdated sentence as the API spec. Fix both documents.

