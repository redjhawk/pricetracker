#!/usr/bin/env bash
set -euo pipefail

APP_USER="pricefollower"
APP_GROUP="pricefollower"
APP_DIR="/opt/pricefollower"
DATA_DIR="/var/lib/pricefollower"
SERVICE_FILE="/etc/systemd/system/pricefollower.service"
SOURCE_BINARY="${1:-}"

usage() {
  echo "Usage: sudo $0 /path/to/pricefollower-binary" >&2
}

if [[ $# -ne 1 ]]; then
  usage
  exit 2
fi

if [[ "$(id -u)" -ne 0 ]]; then
  echo "Error: run this installer as root (for example, with sudo)." >&2
  exit 1
fi

if [[ ! -f "$SOURCE_BINARY" ]]; then
  echo "Error: binary not found: $SOURCE_BINARY" >&2
  exit 1
fi

if ! command -v systemctl >/dev/null 2>&1; then
  echo "Error: systemctl is required to install the service." >&2
  exit 1
fi

if ! getent group "$APP_GROUP" >/dev/null; then
  groupadd --system "$APP_GROUP"
fi

if id "$APP_USER" >/dev/null 2>&1; then
  usermod --gid "$APP_GROUP" --shell /usr/sbin/nologin "$APP_USER"
else
  useradd --system --gid "$APP_GROUP" --home-dir "$DATA_DIR" \
    --no-create-home --shell /usr/sbin/nologin "$APP_USER"
fi

install -d -m 0755 "$APP_DIR"
install -d -m 0750 "$DATA_DIR"
install -m 0755 "$SOURCE_BINARY" "$APP_DIR/pricefollower"
chown -R "$APP_USER:$APP_GROUP" "$APP_DIR" "$DATA_DIR"

cat > "$SERVICE_FILE" <<'UNIT'
[Unit]
Description=PriceFollower
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=pricefollower
Group=pricefollower
Environment=PRICEFOLLOWER_DATA_DIR=/var/lib/pricefollower
Environment=PORT=3001
ExecStart=/opt/pricefollower/pricefollower
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
UNIT

chmod 0644 "$SERVICE_FILE"
systemctl daemon-reload
systemctl enable pricefollower
if systemctl is-active --quiet pricefollower; then
  systemctl restart pricefollower
else
  systemctl start pricefollower
fi

echo "PriceFollower is installed and enabled."
systemctl --no-pager --full status pricefollower || true
