# Issue #265 Settings: stop returning the saved Claude token and LeBoncoin session in full

**Problem:** `GET /api/v1/settings/claude-token`, `GET /api/v1/settings/leboncoin-session` and the `PUT /api/v1/settings` response return the saved Claude token and LeBoncoin session value in full, and Settings shows them in its fields. The saved secrets should never leave the server again once saved.

**Suggested work:**
- The API returns, for each of the two secrets, whether one is saved and a short masked hint (for example the last 4 characters), never the full value. Update `API_SPECIFICATION.md` first.
- Settings shows the masked hint and an empty input to replace the secret.
- **Saving Settings must not change a secret the user did not replace.** Today the frontend compares the field with the loaded value to decide what to send (`src/components/SettingsModal.tsx`), and `PUT /api/v1/settings` treats an empty string as "remove". With masked values, the masked hint must never be sent back or saved as the new value, and an untouched field must leave the saved secret exactly as it is (same value, same revision, no new Claude verification). Saving only the LeBoncoin session must not touch the Claude token, and the reverse.
- Removing a secret stays possible, as an explicit action that cannot be confused with "unchanged".
- The LeBoncoin session revision check (`SESSION_CHANGED`) keeps working.


## Comments by trusted authors

