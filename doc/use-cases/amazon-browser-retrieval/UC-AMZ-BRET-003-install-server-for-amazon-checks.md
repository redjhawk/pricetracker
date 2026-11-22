# UC-AMZ-BRET-003 — Install or update the server so that Amazon checks work

**Primary actor:** Operator who installs and runs the server
**Supporting systems:** Repository installation documentation, host or container, real web browser, PriceFollower application
**Trigger:** The operator installs the server on a new machine or updates an existing installation.
**Origin:** Issue #204: on the installed server every Amazon check returned **Retrieval error** after the browser change.
**Covers:** FR-AMZ-BRET-006, FR-AMZ-BRET-001

## Preconditions

- The server has no web browser installed (or the operator does not know which one is needed).

## Main flow

1. The operator reads the installation documentation in the repository.
2. The documentation names the web browser to install and how the application finds it.
3. The operator installs the browser as documented.
4. When the service runs in a container or as root, the operator applies the settings the documentation gives for that case.
5. The operator starts the server.
6. Amazon checks of available listings succeed (UC-AMZ-BRET-001).

## Alternatives and errors

- **Browser missing or not found at start:** the server log says so at start; Amazon checks fail with "browser could not start" and its reason (UC-AMZ-BRET-002).
- **Update of an existing installation:** the operator follows the same documentation; items stopped before the update follow UC-AMZ-BRET-004.

## Postconditions

- Following only the installation documentation gives a server whose Amazon checks of available listings succeed.
