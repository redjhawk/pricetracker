# Questions for issue #272: data directory and database readable by the service user only

The modes (`0700` directory, `0600` database files), the tightening at start-up and the documentation are specified in [data-file-permissions](../../specifications/data-file-permissions/functional.md). One point needs your answer.

1. **A mode cannot be set at start-up.** Only the owner of a file or directory can change its mode. `deploy/README.md` only requires the service user to be able to *write* the data directory, so an installation may have a data directory owned by another account (for example `root:pricefollower` at `0770`), or database files on a read-only file system. Today such an installation starts normally. When the server, development mode or the `admin-password` command cannot set the data directory to `0700` or a database file to `0600`, what should happen?
   - a) Refuse to start: the program exits with an error naming the path and its current mode, so the operator must fix the ownership or modes before the service runs again.
   - b) Start normally and write a warning to the log naming the path and its current mode; the files stay readable by the group until the operator fixes them.
   - c) Something else (please describe).
