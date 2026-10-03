# LeBoncoin session from Settings

When LeBoncoin answers PriceFollower with a verification challenge (HTTP 403), you can give the application a session that you verified yourself in a browser. You capture it on a desktop computer with a small helper, then paste it into **Settings › LeBonCoin session** in the PriceFollower interface. No file, server shell access, environment variable or restart is needed.

The helper uses a dedicated browser window with a fresh temporary profile: it does not read your personal browser profile and you do not sign in to LeBoncoin. It never clicks or solves a challenge for you; you complete any verification yourself. A session that works on your desktop is not guaranteed to be accepted from the server's device or network, and LeBoncoin can expire or reject it at any time.

## Prerequisites

- A desktop computer with a graphical session (Linux with X11 or Wayland, or macOS). Run the helper from a terminal opened in that desktop session.
- Google Chrome or Chromium installed. The helper finds `google-chrome`, `google-chrome-stable`, `chromium` or `chromium-browser` on your `PATH` (on macOS, Google Chrome in `/Applications`), then Playwright's browser if installed. Pass `--browser <path>` to choose another executable.
- Node.js 20.19 or newer for the helper.
- A copy of this repository with its dependencies installed: run `npm ci` once in the repository.
- An active LeBoncoin France listing URL.

## Capture a session

```bash
node scripts/capture-leboncoin-session.mjs --url 'https://www.leboncoin.fr/ad/voitures/3245888872'
```

The listing above is only an example; use any active LeBoncoin France listing. Options:

- `--browser <path>`: browser executable to open, for example `--browser /usr/bin/google-chrome`.
- `--timeout-seconds <1-3600>`: how long to wait for you (default 600 seconds).
- `--help`: show usage.

What you see:

1. A first message, `PriceFollower LeBoncoin session capture — checking prerequisites…`, immediately.
2. `Opening <browser> with a temporary profile…`, then `A new browser window is open…`. The window can open behind your terminal or editor: if you do not see it, look for a “ready” notification or switch windows (Alt+Tab or Activities).
3. `Waiting for you to complete any LeBoncoin verification in that window. Deadline: HH:MM:SS.` Complete any LeBoncoin verification in that window and leave it open. `Page loaded; checking the listing…` appears each time a LeBoncoin page loads.
4. On success the helper closes its window and prints:

   ```text
   Session captured. Paste the next line into Settings › LeBonCoin session in PriceFollower. Keep it private: do not share it in chats, tickets or logs.
   datadome=<value>
   ```

The `datadome=…` line is the only output on standard output; all messages go to standard error. Press Ctrl+C at any time to cancel.

## Paste it into Settings

1. Copy the whole `datadome=…` line.
2. In PriceFollower, select the profile icon at the top right of the header, then **Settings**.
3. Paste the line into **LeBonCoin session** and save. The application keeps only the datadome value. You can also paste the raw value or a cookie string containing `datadome=…`.

The session is stored in the application's database and applies to the next LeBoncoin price check, without a restart. Saving does not start a check by itself.

## Check the result

Refresh a LeBoncoin item from its details page, or use the main refresh control, then check the attempt status and the newly detected price. A rejected attempt keeps the last successful price.

Reopen **Settings**: if the most recent attempt that used the saved session failed, the modal shows when, and whether LeBoncoin rejected the session. If it says the session is expired or revoked, it is no longer used.

## Renew or clear

- **Renew** when Settings reports a rejection, expiry or revocation, or when LeBoncoin checks fail with a verification challenge: run the helper again and paste the new line into Settings. The application also saves renewed cookies that LeBoncoin sends during successful checks.
- **Clear** by saving an empty **LeBonCoin session** field. LeBoncoin checks then run without a session.

## Keep the value secret

The printed value gives access to your verified LeBoncoin session. It stays in your terminal scrollback and clipboard: do not paste it into chats, tickets, URLs or shared logs. Anyone who can open the PriceFollower interface can read it in Settings.

## Failure messages

Every run ends with one final message and a nonzero exit status on failure; no session value is printed on failure.

| Message starts with | What to do |
| --- | --- |
| `Usage:` | Pass `--url <listing-url>` and only the options listed above. |
| `… is no longer supported. The session is now printed here…` | Remove the old output-file option; the session is printed for pasting into Settings. |
| `Use a supported HTTPS LeBoncoin listing URL.` | Use a `https://www.leboncoin.fr/ad/<category>/<number>` listing URL. |
| `Timeout must be an integer` | Use a whole number of seconds from 1 to 3600. |
| `Node.js 20.19 or newer is required` | Install a newer Node.js. |
| `Repository dependencies are missing.` | Run `npm ci` in the repository. |
| `No graphical display was found` | Run the helper from a terminal inside your desktop session, not over SSH without a display. |
| `This helper supports Linux and macOS desktops only.` | Use a Linux or macOS desktop. |
| `The browser at … was not found` / `No Chrome or Chromium browser was found.` | Install Google Chrome or Chromium, or pass a correct `--browser` path. |
| `The browser exited during startup` / `The browser started but …` | Check that the browser starts normally from this desktop session, or pass another `--browser`. |
| `Browser debugging port collision` | Retry. |
| `The browser window was closed before verification finished.` | Run again and leave the window open until the helper reports success. |
| `The browser was quit before verification finished.` | Run again without quitting the browser. |
| `The browser exited unexpectedly` | Retry; if it repeats, start the browser manually to check it works. |
| `Lost the connection to the browser` | Retry. |
| `Capture timed out after N seconds: …` | The reason says what was last seen: a verification page still shown, a page that is not the requested active listing (check the URL or try another listing), a listing page that could not be loaded, or no page yet. Retry with more time or another listing. |
| `The listing loaded but LeBoncoin did not set a usable datadome cookie.` | Retry, or try another active listing. |
| `Capture cancelled.` | You pressed Ctrl+C or the helper was stopped. |
| `Capture failed because of an unexpected error` | Retry; the message names only the error type. |

The helper always closes its browser and removes its temporary profile. Only a forced kill (SIGKILL) or power loss can leave a `pricefollower-capture-*` directory in your system temporary directory; after checking that no capture is running, remove that directory.

## Helper tests

```bash
node --test scripts/capture-leboncoin-session.test.mjs
```

These tests use synthetic values and simulated browsers. A real capture and a successful check from the server remain separate operator checks.

## Previous file-based workflow

Earlier versions transferred a session file to the server. That file-based workflow was removed; see the [deprecation record](deprecated/leboncoin-session-file.md) for its description and for cleaning up an existing deployment.
