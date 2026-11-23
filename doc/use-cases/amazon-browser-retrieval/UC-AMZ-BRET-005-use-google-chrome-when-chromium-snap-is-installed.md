# UC-AMZ-BRET-005 — Use Google Chrome when a Chromium snap is installed

**Primary actor:** Operator who installs and runs the server
**Supporting systems:** Repository installation documentation, host (usually Ubuntu), Chromium snap, Google Chrome, PriceFollower application
**Trigger:** The operator installs or updates the server on a machine where a Chromium snap is installed, or reads the documentation after Amazon checks fail with "browser could not start".
**Origin:** Issue #226 (review comment R4-01 on PR #206): the application finds the snap Chromium before Google Chrome, and snap confinement can stop it from starting for the service.
**Covers:** FR-AMZ-BRET-008, FR-AMZ-BRET-006

## Preconditions

- A Chromium snap is installed on the server, so `chromium` or `chromium-browser` on the service's `PATH` leads to it.
- `PRICEFOLLOWER_BROWSER_PATH` is not set.

## Main flow

1. The operator reads the section on the browser for Amazon checks in the installation documentation.
2. The section lists the executable names the application looks up on the service's `PATH`, in this order: `chromium`, `chromium-browser`, `google-chrome-stable`, `google-chrome`.
3. Next to that list, one sentence tells the operator: when a Chromium snap is installed (usual on Ubuntu), set `PRICEFOLLOWER_BROWSER_PATH` to the Google Chrome executable, for example `/usr/bin/google-chrome-stable`.
4. The operator installs Google Chrome as documented, if not already installed.
5. The operator sets `PRICEFOLLOWER_BROWSER_PATH` to the Google Chrome executable for the service.
6. Amazon checks use Google Chrome and those of available listings succeed (UC-AMZ-BRET-001).

## Alternatives and errors

- **Only Google Chrome installed, or `PRICEFOLLOWER_BROWSER_PATH` already set:** nothing to change; the operator follows UC-AMZ-BRET-003.
- **Operator does not set the setting:** the application behaves as before issue #226: it uses the snap Chromium, and when snap confinement stops it, every Amazon check fails and the server log gives "browser could not start" with the browser's reason (UC-AMZ-BRET-002). The operator then applies step 3.

## Postconditions

- The documented list of looked-up names is unchanged and matches the application's lookup, in the same order.
- The application's behaviour, including the order in which it looks for a browser, is unchanged.
