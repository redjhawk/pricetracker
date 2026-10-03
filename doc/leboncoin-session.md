# Manually verified LeBoncoin sessions

When LeBoncoin asks for verification, use a dedicated desktop browser to complete it yourself, then transfer the resulting private session file to the Raspberry Pi. This workflow exports only the applicable `datadome` cookie. It does not use your personal browser profile or require a LeBoncoin login. Desktop success does not guarantee acceptance from the Pi's device or network, and the session can expire or be rejected later.

## Capture on your desktop

Use Node.js 22 or newer, the repository dependencies (`npm ci`), and a graphical POSIX desktop with Chrome or Chromium. Windows filesystem permissions are not supported by this helper. If using Playwright's bundled browser, install it with `npx playwright install chromium`; otherwise pass your installed browser's executable through `--browser`.

Create a private directory outside the repository. Its existing parents must be real directories, without symbolic links:

```bash
install -d -m 0700 "$HOME/.pricefollower-session"
node scripts/capture-leboncoin-session.mjs \
  --url 'https://www.leboncoin.fr/ad/voitures/3245888872' \
  --output "$HOME/.pricefollower-session/session.json" \
  --browser /opt/google/chrome/chrome
```

The listing above is an example used during investigation. Substitute any supported active LeBoncoin France listing; the example might later be removed. Omit `--browser` to use the installed Playwright browser. Quote paths containing spaces.

The helper opens an isolated visible browser, visits LeBoncoin and the requested listing, and waits for you to complete any verification. Do not sign in. It confirms a successful page response, matching active listing identity, and an applicable cookie before exporting. It never clicks or solves a challenge. If the page is unavailable, cancel and try another active listing.

The default total deadline is 600 seconds; `--timeout-seconds` accepts integers from 1 through 3600. Ctrl+C cancels. Timeout, cancellation, browser failure, or incomplete verification returns a nonzero exit status and preserves any previous export. Successful renewal replaces the file atomically with mode 0600. Existing symlinks, nonregular targets, unsafe ownership/permissions, and destinations inside the repository are rejected. A successful message names the output file without printing its contents.

Normal exits close the dedicated browser and remove its temporary profile. SIGKILL or power loss can leave private `pricefollower-capture-*` directories under your system temporary directory; after confirming no capture process is running, remove only the directories belonging to that interrupted capture. Do not reuse these as browser profiles. Keep the exported file out of source control, backups shared with others, chat, URLs, and application forms.

## Install the application release

Build and deploy using the existing release procedure, substituting your SSH target:

```bash
./scripts/deploy-armv6.sh pi@raspberry-pi
```

This builds the ARMv6 release and installs the existing `pricefollower` systemd service. For build-only verification use `./scripts/build-release.sh 6`. See [deployment instructions](../DEPLOYMENT.md). No Node.js or browser service is required on the Pi. The session transfer is separate from the application release.

## Transfer and install the session

Use your normal SSH host-key verification and account permissions; do not disable host-key checking. The commands below use the example target `pi@raspberry-pi`. First create a unique mode-0700 staging directory owned by your SSH account on the Pi. Run these commands in the same local shell:

```bash
LBC_TARGET=pi@raspberry-pi
LBC_STAGE=$(ssh "$LBC_TARGET" 'umask 077; mktemp -d "$HOME/.pricefollower-session-transfer.XXXXXXXX"')
# Check that the returned absolute path contains only safe shell characters.
case "$LBC_STAGE" in
  /*) ;;
  *) echo 'Invalid staging path' >&2; exit 1 ;;
esac
case "$LBC_STAGE" in
  *[!a-zA-Z0-9_./-]*) echo 'Unsupported staging path' >&2; exit 1 ;;
esac
scp "$HOME/.pricefollower-session/session.json" "$LBC_TARGET:$LBC_STAGE/session.json"
```

The protected staging directory prevents access by other local users even during transfer. Next, open an interactive SSH shell in that staging directory:

```bash
ssh -t "$LBC_TARGET" "cd -- '$LBC_STAGE' && exec bash"
```

In that remote shell, run the following block. It installs through a temporary file on the final filesystem and renames atomically. Enter your normal sudo password if prompted; the cookie itself is never a command argument:

```bash
sudo bash -s -- "$PWD" <<'REMOTE'
set -eu
stage=$1
source_file="$stage/session.json"
target_dir=/var/lib/pricefollower/leboncoin
install -d -o pricefollower -g pricefollower -m 0700 "$target_dir"
umask 077
private_file=$(mktemp "$target_dir/.session-install.XXXXXXXX")
trap 'rm -f -- "$private_file"' EXIT
install -o pricefollower -g pricefollower -m 0600 "$source_file" "$private_file"
mv -fT -- "$private_file" "$target_dir/session.json"
rm -f -- "$source_file"
rmdir -- "$stage"
REMOTE
exit
```

If an installation command fails, retain your desktop export, resolve the reported permission/setup issue, and retry. Delete any unused remote staging directory through SSH after confirming it belongs to this transfer. Never use a world-readable temporary copy. This setup supports one running PriceFollower process owning the session directory.

## Enable once, then renew without restarting

On the Pi, create a systemd drop-in:

```bash
sudo install -d -m 0755 /etc/systemd/system/pricefollower.service.d
sudo tee /etc/systemd/system/pricefollower.service.d/leboncoin-session.conf >/dev/null <<'UNIT'
[Service]
Environment=LEBONCOIN_SESSION_FILE=/var/lib/pricefollower/leboncoin/session.json
UNIT
sudo systemctl daemon-reload
sudo systemctl restart pricefollower
sudo systemctl is-active pricefollower
```

The application installer preserves this drop-in. The service keeps verified cookie updates in the private companion file `session.json.state.json`. That file belongs to the collector: do not copy an old sidecar over a newer import or edit it manually. Keep both files in the service-owned mode-0700 directory.

If diagnostics report a malformed, unreadable, or unsafe sidecar, replacing the import alone does not repair that sidecar. On the Pi, stop the service (`sudo systemctl stop pricefollower`), check the ownership and modes with `sudo ls -ld /var/lib/pricefollower/leboncoin /var/lib/pricefollower/leboncoin/session.json.state.json` without printing contents, and remove only the broken collector sidecar with `sudo rm -- /var/lib/pricefollower/leboncoin/session.json.state.json`. Install a freshly verified import using the procedure above before starting the service again (`sudo systemctl start pricefollower`); do not deliberately restore an old revoked session by deleting its state. Restore directory ownership to `pricefollower:pricefollower` and mode 0700 if needed.

Open your existing PriceFollower interface, select LeBoncoin, and refresh the listing from its details page, or use the main refresh control. Check the attempt status and newly detected price. A saved file or a running service alone does not establish successful server collection. On a rejection/403, the previous successful price remains available and the attempt records the existing collection error. Safe service diagnostics can be read with `sudo journalctl -u pricefollower --since '10 minutes ago'`; do not print session-file contents.

For renewal, repeat desktop capture and the protected transfer/install sequence. The collector reads the replacement on the next collection attempt; no service restart is needed. If the renewed session is rejected from the Pi, repeat human verification as needed; neither portability nor lifetime is guaranteed.

To disable session-assisted collection, remove `/etc/systemd/system/pricefollower.service.d/leboncoin-session.conf`, run `sudo systemctl daemon-reload`, and restart `pricefollower`. Remove private session files when no longer needed.

## Focused helper checks

```bash
node --test scripts/capture-leboncoin-session.test.mjs
```

These tests use synthetic session values, private temporary directories, and mocked browser/process interfaces. Live desktop verification and successful Pi collection remain separate operator checks.
