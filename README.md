# srvdog

Lightweight server resource dashboard for small Linux VPS hosts.

## What It Does

- Binds to `127.0.0.1:8090`
- Intended to be accessed through an SSH tunnel
- Keeps low-frequency background history
- Switches to high-frequency realtime sampling while the page is open
- Shows:
  - CPU
  - load average
  - memory
  - swap
  - root disk usage
  - network realtime throughput
  - network rolling average throughput
  - Docker container summary
- Supports:
  - retention-day changes
  - export as JSON or CSV
  - clearing sampling history
  - a local-only `Clash` management tab for subscription operations

## Access Model

The app is intentionally local-only on the server.

Open an SSH tunnel:

```bash
ssh -L 8090:127.0.0.1:8090 root@107.174.48.241 -p 45678
```

Then open:

```text
http://127.0.0.1:8090
```

## Launcher Scripts

Convenience launchers are available in `scripts/`:

- `scripts/open-srvdog.ps1`
- `scripts/open-srvdog.sh`

They:

- check whether local port `8090` is already in use
- open the SSH tunnel
- launch the dashboard in the default browser

See [scripts/README.md](scripts/README.md) for usage details.

## Local Build On Windows For Linux

```powershell
$env:GOOS="linux"
$env:GOARCH="amd64"
$env:CGO_ENABLED="0"
& "C:\Program Files\Go\bin\go.exe" build -o srvdog-linux-amd64 ./cmd/srvdog
```

## Test

```powershell
& "C:\Program Files\Go\bin\go.exe" test ./...
```

## Run On Linux Server

```bash
chmod +x srvdog-linux-amd64
./srvdog-linux-amd64
```

The service writes runtime data to:

- `data/settings.json`
- `data/history/*.jsonl`
- `data/clash/config.draft.yaml`
- `data/clash/script.draft.yaml`
- `data/clash/operations.log`

## Clash Tab

The `Clash` tab is intended for operators who already access `srvdog` through the SSH tunnel. It does not add a separate login layer.

Features:

- view current token and published subscription URLs
- edit and save config YAML drafts
- validate and publish config updates
- edit and publish the managed dynamic script block
- trigger geodata updates
- rotate the token immediately and invalidate the old public directory
- inspect operation logs and geodata update logs

## Clash Environment

These environment variables can override the built-in defaults:

- `SRVDOG_CLASH_TOKEN_FILE`
- `SRVDOG_CLASH_SITE_DIR`
- `SRVDOG_CLASH_PUBLIC_BASE_URL`
- `SRVDOG_CLASH_GEODATA_SCRIPT`
- `SRVDOG_CLASH_GEODATA_LOG_PATH`
- `SRVDOG_CLASH_MIHOMO_IMAGE`

Default assumptions match the current VPS layout:

- token file at `/root/mihomo-subscription/token`
- published subscription directory at `/opt/cypht/data/site-wg`
- public subscription base URL at `http://107.174.48.241/wg`
- geodata update script at `/usr/local/bin/update-mihomo-geodata.sh`
- geodata log at `/var/log/update-mihomo-geodata.log`
- validation image `docker.io/metacubex/mihomo:Alpha`

## Sampling Model

- Low-frequency background sampling:
  - every 5 minutes
- High-frequency realtime sampling:
  - every 2 seconds while an active browser is sending heartbeats
- Docker summary refresh:
  - every 30 seconds

## History

- Default retention: 7 days
- Retention can be changed in the page
- Old history files are cleaned up automatically

## Export And Clear

- Export supports JSON and CSV
- Export includes low-frequency history and the current realtime buffer
- Clear history removes stored sampling history and the in-memory realtime buffer

## Notes

- The runtime target is Linux
- The code is compiled on Windows and deployed to Linux
- Docker summary collection depends on the `docker` CLI being available on the server
