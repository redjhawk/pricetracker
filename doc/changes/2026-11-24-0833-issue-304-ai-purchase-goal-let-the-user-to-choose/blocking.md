# Blocking review comments (round 2)

## R2-01 `API_SPECIFICATION.md:499`

The contract and the implementation now disagree on the order. After the R1-02 fix, `compareGoals` in `internal/service/purchase_goal.go:28` sorts in three steps: first by base letters (lower case, Latin diacritics mapped to their base letter, `æ`/`œ`/`ß` expanded), then by the lower-case text, then by the exact text. This 'Order' bullet still specifies only two steps: lower case, then byte order. Under the published contract, 'écran' must sort after 'zoom', but the server returns it between 'dessin' and 'zoom'. `technical.md` (API section) says this bullet 'still needs this text' because the developer stage could not edit it. Fix: replace the bullet with the three-step rule from `doc/specifications/purchase-goal-choice/technical.md` line 32, including the écran/ecran/Gaming examples, so `API_SPECIFICATION.md` stays the source of truth for the contract.

Triage: Confirmed contract violation: API_SPECIFICATION.md:499 still specifies a two-step order (lower case, then byte order), but compareGoals in internal/service/purchase_goal.go:28-35 sorts by base letters first, so the server returns "écran" between "dessin" and "zoom" where the published contract puts it after "zoom". technical.md line 32 itself records that the API bullet still needs the three-step text.

