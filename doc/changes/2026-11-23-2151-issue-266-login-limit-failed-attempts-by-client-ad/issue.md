# Issue #266 Login: limit failed attempts by client address instead of locking the username

**Problem:** Failed logins are counted per username (`internal/service/auth.go`, `Login`): after 5 failures the username is locked for 1 minute for everyone, so repeated wrong attempts by anyone keep a user or the administrator out. Password checks also run one at a time (`hashSlot`) with no limit per client, so many attempts make every other login wait.

The application is only reachable through Cloudflare, which sends the client address in the `CF-Connecting-IP` header.

**Suggested work:**
- Count failed logins per client address and username instead of per username alone, so a client that fails is the only one refused. Take the client address from `CF-Connecting-IP` when present, else from the connection.
- Refuse a client over its limit before the password hash is computed.
- Answer `429` at once when too many password checks are already waiting, instead of queueing without bound.
- Keep the memory of failures bounded, as today.
- Update `API_SPECIFICATION.md` and the functional specification for the changed lock rule.


## Comments by trusted authors

