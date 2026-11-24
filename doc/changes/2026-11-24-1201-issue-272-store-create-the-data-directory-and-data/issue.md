# Issue #272 Store: create the data directory and database files readable by the service user only

**Problem:** `store.Open` (`internal/store/store.go`) creates the data directory with mode `0750`, and the SQLite files get the default mode. The database holds password hashes, the Claude token and the LeBoncoin session, so other accounts of the service user's group can read them.

**Suggested work:** Create the data directory with `0700` and make sure the SQLite database files (including the `-wal` and `-shm` files) are `0600`. Tighten the modes of an existing data directory and database at start-up as well. Mention the modes in `deploy/README.md`.


## Comments by trusted authors

