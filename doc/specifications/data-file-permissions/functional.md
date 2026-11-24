# Functional specification: Data directory and database readable by the service user only

Status: specified (issue #272); FR-DATA-PERM-008 is pending the answer to question 1 of the issue #272 change folder
Owner: functional specification agent
User decision/reference: GitHub issue #272 ("Store: create the data directory and database files readable by the service user only").

## Purpose and scope

The data directory (`PRICEFOLLOWER_DATA_DIR`) is created with mode `0750`, and the SQLite database files get the default mode, so accounts in the service user's group can read them. The database holds password hashes, Claude tokens and LeBoncoin sessions. The user asks that the data directory be `0700` and the database files `0600`, that the modes of an existing installation be tightened at start-up, and that the installation documentation state these modes.

Actors: the server operator (installs and starts the application as the service user, runs the `admin-password` command), other accounts on the server (in the service user's group or not).

Included: the mode of the data directory; the mode of the database files in it: `pricefollower.sqlite` and its SQLite companion files `pricefollower.sqlite-wal` and `pricefollower.sqlite-shm`; every program start that opens the database (the server, development mode and the `admin-password` command); the server setup section of `deploy/README.md`.

Excluded: other files and directories inside the data directory (the Amazon browser profile `amazon-browser/`, `.config/`, `.cache/` or files the operator placed there) keep their own modes; the data directory at `0700` already makes them unreachable to other accounts. File owners and groups are not changed. Database content, screens, API and messages of the application are unchanged.

Terms:

- **Owner-only directory:** mode `0700` (read, write and enter for the owner; nothing for group and others).
- **Owner-only file:** mode `0600` (read and write for the owner; nothing for group and others).
- **Database files:** `pricefollower.sqlite`, `pricefollower.sqlite-wal` and `pricefollower.sqlite-shm` in the data directory, whichever exist.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-DATA-PERM-001 | A program start that opens the database finds no data directory. | The data directory is created as an owner-only directory, whatever the process umask. Missing parent directories are created as before. | Start the server with `PRICEFOLLOWER_DATA_DIR` set to a missing path and umask `0002` or `0022`: `stat -c %a <data directory>` prints `700`. |
| FR-DATA-PERM-002 | The database file does not exist when the program starts. | It is created as an owner-only file, whatever the process umask. | After a first start with umask `0002` or `0022`, `stat -c %a <data directory>/pricefollower.sqlite` prints `600`. |
| FR-DATA-PERM-003 | The program writes to the database while running. | The `-wal` and `-shm` files it creates are owner-only files, whatever the process umask. | While the server runs after adding an item, `stat -c %a` of `pricefollower.sqlite-wal` and `pricefollower.sqlite-shm` prints `600` for each file present. |
| FR-DATA-PERM-004 | A program start finds an existing data directory with a mode other than `0700` (for example `0750`, `0755`, `0770`), owned by the service user. | Before the database is opened, the data directory's mode is set to `0700`. | With a data directory at `0750` from an earlier release, after a start `stat -c %a <data directory>` prints `700`. |
| FR-DATA-PERM-005 | A program start finds existing database files with a mode other than `0600` (for example `0644`, `0664`), owned by the service user. | Before the database is used, each existing database file's mode is set to `0600`. | With `pricefollower.sqlite`, `-wal` and `-shm` at `0644` (or `0664`), after a start `stat -c %a` prints `600` for each of them. |
| FR-DATA-PERM-006 | Any start that applies FR-DATA-PERM-001 to 005. | The application starts and works as before: items, prices, users, sessions, Claude tokens and LeBoncoin sessions are unchanged, and the data directory and database files keep their owner and group. Contents of the data directory other than the database files keep their modes. | After upgrading an installation with a `0750` directory and `0644` database, the operator logs in as before and sees the same items and settings; `amazon-browser/` has the same mode as before the start; `stat -c %U:%G` of the directory and database files is unchanged. |
| FR-DATA-PERM-007 | The data directory and database files are already owner-only. | Starting changes nothing and works as before. | Two consecutive starts both leave `700` and `600`; the application works. |
| FR-DATA-PERM-008 | A mode required by FR-DATA-PERM-004 or 005 cannot be set (for example the data directory belongs to another account and the service user only has group write access, or the file system is read-only). | Pending: depends on question 1 (`doc/changes/2026-11-24-1201-issue-272-store-create-the-data-directory-and-data/questions.md`). | Pending. |
| FR-DATA-PERM-009 | The operator reads the server setup section of `deploy/README.md`. | It states that the application creates the data directory with mode `0700` and the database files (`pricefollower.sqlite`, `-wal`, `-shm`) with mode `0600`, that it sets these modes on an existing data directory and database at each start, and that other accounts, including the service user's group, therefore cannot read them. | The server setup section of `deploy/README.md` contains `0700`, `0600` and the three database file names, and says the modes are applied at start-up. |

## States and corner cases

- **Umask:** the resulting modes do not depend on the umask of the process (FR-DATA-PERM-001 to 003).
- **`admin-password` command and development mode:** they open the database like the server, so FR-DATA-PERM-001 to 007 apply to their data directory too.
- **Missing companion files:** a `-wal` or `-shm` file absent at start-up is not created for this purpose; it gets its mode when the program creates it (FR-DATA-PERM-003).
- **More restrictive modes:** a data directory or database file with a mode other than exactly `0700` / `0600` is set to exactly that mode, so the program can still read and write it.
- No screen, message, setting or loading state of the application changes.

## Acceptance criteria summary

- A fresh installation has a `0700` data directory and `0600` database files, whatever the umask.
- An existing installation's data directory and database files become `0700` and `0600` at the next start, with data, owner and group unchanged.
- Other contents of the data directory keep their modes.
- `deploy/README.md` documents the modes.

## Open questions

1. Behaviour when a mode cannot be set at start-up (FR-DATA-PERM-008): see `doc/changes/2026-11-24-1201-issue-272-store-create-the-data-directory-and-data/questions.md`.
